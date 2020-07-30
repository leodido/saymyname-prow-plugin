package main

import (
	"encoding/json"
	"flag"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"

	"k8s.io/test-infra/pkg/flagutil"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/config/secret"
	prowflagutil "k8s.io/test-infra/prow/flagutil"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/interrupts"
	"k8s.io/test-infra/prow/pjutil"
	"k8s.io/test-infra/prow/pluginhelp"
	"k8s.io/test-infra/prow/pluginhelp/externalplugins"
	"k8s.io/test-infra/prow/plugins"
)

const pluginName = "saymyname"

var (
	command   = "/poiana"
	commandRe = regexp.MustCompile(`(?mi)^` + command + `\s*$`)
	replies   = []string{
		"**I'm [poiana](https://github.com/poiana), I stop the drama!**",
		"I'm [poiana](https://github.com/poiana), I fly around Falco projects.",
		"I'm [poiana](https://github.com/poiana), for issue with Kubernetes ask to [fntlnz](https://github.com/fntlnz).",
		"I'm [poiana](https://github.com/poiana), I'm here to look at [Kris](https://github.com/kris-nova)'s commit messages...",
		"I'm [poiana](https://github.com/poiana), I obey my falconer [leodido](https://github.com/leodido).",
		"**[Poiana](https://github.com/poiana) stops the drama!**",
	}
)

type options struct {
	port       int
	dryRun     bool
	github     prowflagutil.GitHubOptions
	hmacSecret string
}

func (o *options) Validate() error {
	for _, group := range []flagutil.OptionGroup{&o.github} {
		if err := group.Validate(o.dryRun); err != nil {
			return err
		}
	}

	return nil
}

func newOptions() *options {
	o := options{}
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	fs.IntVar(&o.port, "port", 8787, "Port to listen on.")
	fs.BoolVar(&o.dryRun, "dry-run", true, "Dry run for testing (uses API tokens but does not mutate).")
	fs.StringVar(&o.hmacSecret, "hmac", "/etc/webhook/hmac", "Path to the file containing the GitHub HMAC secret.")

	for _, group := range []flagutil.OptionGroup{&o.github} {
		group.AddFlags(fs)
	}
	fs.Parse(os.Args[1:])

	return &o
}

func helpProvider(enabledRepos []config.OrgRepo) (*pluginhelp.PluginHelp, error) {
	pluginHelp := &pluginhelp.PluginHelp{
		Description: `This is a Prow external plugin. If you comment "` + command + `" on Github, Prow bot replies properly.`,
	}
	pluginHelp.AddCommand(pluginhelp.Command{
		Usage:       command,
		Description: "Tells you who it is!",
		WhoCanUse:   "Anyone",
		Examples:    []string{command},
	})
	return pluginHelp, nil
}

func main() {
	o := newOptions()
	if err := o.Validate(); err != nil {
		logrus.Fatalf("Invalid options: %v", err)
	}

	logrus.SetFormatter(&logrus.JSONFormatter{})
	// todo(leodido) > use global option from the Prow config.
	logrus.SetLevel(logrus.DebugLevel)
	log := logrus.StandardLogger().WithField("plugin", pluginName)

	secretAgent := &secret.Agent{}
	if err := secretAgent.Start([]string{o.github.TokenPath, o.hmacSecret}); err != nil {
		logrus.WithError(err).Fatal("Error starting secrets agent.")
	}

	githubClient, err := o.github.GitHubClient(secretAgent, o.dryRun)
	if err != nil {
		logrus.WithError(err).Fatal("Error getting GitHub client.")
	}

	serv := &server{
		tokenGenerator: secretAgent.GetTokenGenerator(o.hmacSecret),
		gh:             githubClient,
		log:            log,
	}

	health := pjutil.NewHealth()

	log.Info("Complete serve setting")
	mux := http.NewServeMux()
	mux.Handle("/", serv)
	externalplugins.ServeExternalPluginHelp(mux, log, helpProvider)
	httpServer := &http.Server{Addr: ":" + strconv.Itoa(o.port), Handler: mux}

	health.ServeReady()

	defer interrupts.WaitForGracefulShutdown()
	interrupts.ListenAndServe(httpServer, 5*time.Second)
}

type server struct {
	tokenGenerator func() []byte
	configAgent    *config.Agent
	gh             github.Client
	log            *logrus.Entry
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	eventType, eventGUID, payload, ok, _ := github.ValidateWebhook(w, r, s.tokenGenerator)
	if !ok {
		return
	}
	// Event received, handle it
	if err := s.handleEvent(eventType, eventGUID, payload); err != nil {
		logrus.WithError(err).Error("Error parsing event.")
	}
}

func (s *server) handleEvent(eventType, eventGUID string, payload []byte) error {
	l := s.log.WithFields(logrus.Fields{
		"event-type":     eventType,
		github.EventGUID: eventGUID,
	})

	switch eventType {
	case "issue_comment":
		var ic github.IssueCommentEvent
		if err := json.Unmarshal(payload, &ic); err != nil {
			return err
		}
		go func() {
			if err := s.handleIssueComment(l, ic); err != nil {
				s.log.WithError(err).WithFields(l.Data).Info("Error handling event.")
			}
		}()
	default:
		logrus.Debugf("skipping event of type %q", eventType)
	}
	return nil
}

func (s *server) handleIssueComment(l *logrus.Entry, ic github.IssueCommentEvent) error {
	l = s.log.WithFields(logrus.Fields{
		"body": ic.Comment.Body,
	})
	l.Info("handleIssueComment called")
	if ic.Action != github.IssueCommentActionCreated || ic.Issue.State == "closed" {
		return nil
	}

	org := ic.Repo.Owner.Login
	repo := ic.Repo.Name
	num := ic.Issue.Number

	l = l.WithFields(logrus.Fields{
		github.OrgLogField:  org,
		github.RepoLogField: repo,
		github.PrLogField:   num,
	})

	if !commandRe.MatchString(ic.Comment.Body) {
		return nil
	}

	reply := replies[rand.Intn(len(replies))]

	l.Infof("Commenting with \"%s\".", reply)
	return s.gh.CreateComment(org, repo, num, plugins.FormatICResponse(ic.Comment, reply))
}

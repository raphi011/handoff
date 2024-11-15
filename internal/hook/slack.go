package hook

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/raphi011/handoff/internal/model"
	"github.com/slack-go/slack"
)

// SlackHook supports sending messages to slack channels that inform on
// test runs.
type SlackHook struct {
	api             *slack.Client
	notifyChannelID string

	log *slog.Logger
}

func NewSlackHook(channelID, token string, log *slog.Logger) *SlackHook {
	return &SlackHook{
		api:             slack.New(token),
		notifyChannelID: channelID,
		log:             log,
	}
}

func (h *SlackHook) Name() string {
	return "Slack"
}

func (h *SlackHook) Init() error {
	_, err := h.api.AuthTest()
	if err != nil {
		return fmt.Errorf("invalid auth token: %w", err)
	}

	return nil
}

func (h *SlackHook) TestSuiteFinishedAsync(suite model.TestSuite, tsr model.TestSuiteRun, callback func(context map[string]any)) {
	if tsr.Result != model.ResultFailed {
		return
	}

	testList := strings.Builder{}

	testList.WriteString(fmt.Sprintf("Test suite run <http://localhost:1337/suites/%[1]s/runs/%[2]d|%[1]s: %[2]d> failed.", suite.Name, tsr.ID))
	testList.WriteString("\n\n")
	testList.WriteString("Results:\n")

	for _, tr := range tsr.LatestTestAttempts() {
		testList.WriteString(fmt.Sprintf("- %s (%s)\n", tr.Name, tr.Result))
	}

	newMarkdownSection := slack.NewSectionBlock(
		slack.NewTextBlockObject(
			"mrkdwn",
			testList.String(),
			false, false,
		),

		nil, nil)

	msg := []slack.MsgOption{
		slack.MsgOptionBlocks(newMarkdownSection),
	}

	_, _, err := h.api.PostMessage(h.notifyChannelID, msg...)
	if err != nil {
		h.log.Error("unable to send slack message", "error", err)
	}
}

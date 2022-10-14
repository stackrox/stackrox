package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/GoogleCloudPlatform/testgrid/metadata/junit"
	"github.com/slack-go/slack"
	"log"
	"os"
	"regexp"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatal("usage: junit2slack <junit report file name>")
	}

	fileName := os.Args[1]
	if _, err := os.Stat(fileName); err == nil {
		data, err := os.ReadFile(fileName)
		if err != nil {
			log.Fatalf("error while reading %s: %s", fileName, err)
		}

		junitSuites, err := junit.Parse(data)
		if err != nil {
			log.Fatalf("error while parsing junit suites in %s: %s", fileName, err)
		}

		slackMsg := convertJunitToSlack(junitSuites)
		b, err := json.Marshal(slackMsg)
		fmt.Println(b)
	} else if errors.Is(err, os.ErrNotExist) {
		log.Fatalf("%s doesn't exist: %s", fileName, err)
	} else {
		log.Fatalf("error while trying to find %s: %s", fileName, err)
	}
}

func convertJunitToSlack(suites *junit.Suites) slack.Msg {
	var failedTestsBlocks []slack.Block
	var attachments []slack.Attachment

	for _, suite := range suites.Suites {
		// We currently only care about failures
		if suite.Failures == 0 {
			continue
		}

		for _, result := range suite.Results {
			// We currently only care about failures
			if result.Failure == nil {
				continue
			}

			var title string
			if result.ClassName == "" {
				title = result.Name
			} else {
				title = fmt.Sprintf("%s: %s", result.ClassName, result.Name)
			}

			titleTextBlock := slack.NewTextBlockObject("plain_text", title, false, false)
			titleSectionBlock := slack.NewSectionBlock(titleTextBlock, nil, nil)
			failedTestsBlocks = append(failedTestsBlocks, titleSectionBlock)

			failureMessage := result.Failure.Message
			// Double the whitespace if more than one space is present because Slack doesn't render it properly
			var re = regexp.MustCompile(`(\s{2,})`)
			failureMessage = re.ReplaceAllString(failureMessage, `$1$1`)

			// Slack has a 3000-character limit for (non-field) text objects
			if len(failureMessage) > 3000 {
				failureMessage = failureMessage[:3000]
			}
			failureMessageTextBlock := slack.NewTextBlockObject("plain_text", failureMessage, false, false)
			failureMessageSectionBlock := slack.NewSectionBlock(failureMessageTextBlock, nil, nil)

			// Add some formatting to the failure title
			failureTitleTextBlock := slack.NewTextBlockObject("mrkdwn",
				fmt.Sprintf("Junit failure message for *%s*", title), false, false)
			failureTitleSectionBlock := slack.NewSectionBlock(failureTitleTextBlock, nil, nil)

			failureAttachment := slack.Attachment{
				Color: "#bb2124",
				Blocks: slack.Blocks{BlockSet: []slack.Block{
					failureTitleSectionBlock,
					failureMessageSectionBlock,
				}},
			}
			attachments = append(attachments, failureAttachment)
		}
	}

	if failedTestsBlocks == nil || len(failedTestsBlocks) <= 0 {
		return slack.Msg{}
	}

	headerTextBlock := slack.NewTextBlockObject("plain_text", "Failed tests", false, false)
	headerBlock := slack.NewHeaderBlock(headerTextBlock)
	failedTestsBlocks = append([]slack.Block{headerBlock}, failedTestsBlocks...)

	failedTestsAttachment := slack.Attachment{
		Color:  "#bb2124",
		Blocks: slack.Blocks{BlockSet: failedTestsBlocks},
	}
	attachments = append([]slack.Attachment{failedTestsAttachment}, attachments...)
	m := slack.Msg{Attachments: attachments}

	return m
}

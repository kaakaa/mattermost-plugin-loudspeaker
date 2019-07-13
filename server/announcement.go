package main

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-server/model"
)

type announcement struct {
	message string
	userID  string
	results []announcementResult
}

type announcementResult struct {
	resultType resultType
	team       *model.Team
	channel    *model.Channel
}

type resultType int

const (
	resultSuccess resultType = iota
	resultErrorNotFoundDefaultChannel
	resultErrorCreatePost
)

func (a *announcement) getResultTable() string {
	if len(a.results) == 0 {
		return "I couldn't find any teams..."
	}

	var ret []string
	ret = append(ret, `| Team | Channel | Result |`)
	ret = append(ret, `|:--|:--|:--|`)
	for _, v := range a.results {
		ret = append(ret, fmt.Sprintf(`| %s | %s | %s |`, v.team.DisplayName, v.channel.DisplayName, v.resultType.toString()))
	}
	return strings.Join(ret, "\n")
}

func (rt resultType) toString() string {
	switch rt {
	case resultSuccess:
		return ":white_check_mark: Success to create message"
	case resultErrorNotFoundDefaultChannel:
		return ":warning: Failed to create post"
	case resultErrorCreatePost:
		return ":warning: Failed to find DEFAULT CHANNEL"
	default:
		return ":skull_and_crossbone: UNKNOWN ERROR"
	}
}

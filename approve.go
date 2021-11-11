package main

import (
	"fmt"
	"regexp"

	libconfig "github.com/opensourceways/community-robot-lib/config"
	"github.com/opensourceways/community-robot-lib/giteeclient"
	"github.com/sirupsen/logrus"
)

const (
	approvedLabel           = "approved"
	approvedAddedMsg        = `***approved*** is added in this pull request by: ***%s***. :wave: `
	approvedNoPermissionMsg = `***%s*** has no permission to %s ***approved*** in this pull request. :astonished:
please contact to the collaborators in this repository.`
	approvedRemovedMsg = `***approved*** is removed in this pull request by: ***%s***. :flushed: `
)

var (
	regAddApprove    = regexp.MustCompile(`(?mi)^/approve\s*$`)
	regRemoveApprove = regexp.MustCompile(`(?mi)^/approve cancel\s*$`)
)

func (bot *robot) handleApprove(e giteeclient.PRNoteEvent, pc libconfig.PluginConfig, log *logrus.Entry) error {
	org, repo := e.GetOrgRep()

	cfg, err := bot.getConfig(pc, org, repo)
	if err != nil {
		return err
	}

	if regAddApprove.MatchString(e.GetComment()) {
		return bot.execAddApproveCommand(cfg, e, log)
	}

	if regRemoveApprove.MatchString(e.GetComment()) {
		return bot.execRemoveApproveCommand(cfg, e, log)
	}

	return nil
}

func (bot *robot) execAddApproveCommand(cfg *botConfig, e giteeclient.PRNoteEvent, log *logrus.Entry) error {
	prInfo := e.GetPRInfo()
	commenter := e.GetCommenter()

	v, err := bot.hasPermission(commenter, prInfo, cfg, log)
	if err != nil {
		return err
	}

	if !v {
		comment := fmt.Sprintf(approvedNoPermissionMsg, commenter, "add")

		return bot.cli.CreatePRComment(prInfo.Org, prInfo.Repo, prInfo.Number, comment)
	}

	if err := bot.cli.AddPRLabel(prInfo.Org, prInfo.Repo, prInfo.Number, approvedLabel); err != nil {
		return err
	}

	err = bot.cli.CreatePRComment(prInfo.Org, prInfo.Repo, prInfo.Number, fmt.Sprintf(approvedAddedMsg, commenter))
	if err != nil {
		log.Error(err)
	}

	return bot.tryMergePR(e, true, cfg, log)
}

func (bot *robot) execRemoveApproveCommand(cfg *botConfig, e giteeclient.PRNoteEvent, log *logrus.Entry) error {
	prInfo := e.GetPRInfo()
	commenter := e.GetCommenter()

	v, err := bot.hasPermission(commenter, prInfo, cfg, log)
	if err != nil {
		return err
	}

	if !v {
		comment := fmt.Sprintf(approvedNoPermissionMsg, commenter, "remove")

		return bot.cli.CreatePRComment(prInfo.Org, prInfo.Repo, prInfo.Number, comment)
	}

	if err := bot.cli.RemovePRLabel(prInfo.Org, prInfo.Repo, prInfo.Number, approvedLabel); err != nil {
		return err
	}

	return bot.cli.CreatePRComment(prInfo.Org, prInfo.Repo, prInfo.Number, fmt.Sprintf(approvedRemovedMsg, commenter))
}

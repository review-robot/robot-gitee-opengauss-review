package main

import (
	"fmt"
	"regexp"
	"strings"

	sdk "gitee.com/openeuler/go-gitee/gitee"
	"github.com/opensourceways/community-robot-lib/giteeclient"
	"github.com/opensourceways/community-robot-lib/utils"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	nonRequiringLabels  = "Labels [**%s**] need to be added."
	nonMissingLabels    = "Labels [**%s**] need to be removed."
	cannotMerge         = "This pull request can not be merged, you can try it again when label requirement meets.\n:astonished: %s"
	prConflict          = "PR conflicts cannot be automatically merged, please resolve conflicts first"
	notEnoughLGTMLabels = "PR needs %d lgtm labels, now only %d"
)

var regCheckPr = regexp.MustCompile(`(?mi)^/check-pr\s*$`)

func (bot *robot) handleCheckPR(e *sdk.NoteEvent, cfg *botConfig, log *logrus.Entry) error {
	ne := giteeclient.NewPRNoteEvent(e)

	if !ne.IsPullRequest() || !ne.IsPROpen() || !ne.IsCreatingCommentEvent() {
		return nil
	}

	if regCheckPr.MatchString(ne.GetComment()) {
		return bot.mergePR(ne.GetPRInfo(), true, cfg, log)
	}

	return nil
}

func (bot *robot) mergePRByLabelChanged(e *sdk.PullRequestEvent, cfg *botConfig, log *logrus.Entry) error {
	if giteeclient.GetPullRequestAction(e) != giteeclient.PRActionUpdatedLabel {
		return nil
	}

	return bot.mergePR(giteeclient.GetPRInfoByPREvent(e), false, cfg, log)
}

func (bot *robot) mergePR(info giteeclient.PRInfo, addComment bool, cfg *botConfig, log *logrus.Entry) error {
	org, repo, number := info.Org, info.Repo, info.Number

	pr, err := bot.cli.GetGiteePullRequest(org, repo, number)
	if err != nil {
		return err
	}

	if err := canMerge(pr, cfg); err != nil {
		log.Infof("merge PR %s/%s/pull/%d failed: %s", org, repo, number, err.Error())

		if !addComment {
			return nil
		}

		return bot.cli.CreatePRComment(org, repo, number, fmt.Sprintf(cannotMerge, err.Error()))
	}

	if pr.TestersNumber > 0 || pr.AssigneesNumber > 0 {
		if err := bot.resetReviewerTesterCount(org, repo, number); err != nil {
			return err
		}
	}

	op := sdk.PullRequestMergePutParam{
		MergeMethod: string(cfg.MergeMethod),
	}

	return bot.cli.MergePR(org, repo, number, op)
}

func (bot *robot) resetReviewerTesterCount(org, repo string, number int32) error {
	reset := int32(0)
	param := sdk.PullRequestUpdateParam{
		AssigneesNumber: &reset,
		TestersNumber:   &reset,
	}

	_, err := bot.cli.UpdatePullRequest(org, repo, number, param)

	return err
}

func canMerge(pr sdk.PullRequest, cfg *botConfig) error {
	if !pr.Mergeable {
		return fmt.Errorf(prConflict)
	}

	ls := labelsToSets(pr.Labels)

	if cfg.LgtmCountsRequired > 1 {
		labelCount := uint(len(getLGTMLabelsOnPR(ls)))
		if labelCount < cfg.LgtmCountsRequired {
			return fmt.Errorf(notEnoughLGTMLabels, cfg.LgtmCountsRequired, labelCount)
		}
	}

	needs := sets.NewString(cfg.RequiringLabels...)
	if cfg.LgtmCountsRequired == 1 && !needs.Has(lgtmLabel) {
		needs.Insert(lgtmLabel)
	}

	misses := sets.NewString(cfg.MissingLabels...)
	mErr := utils.NewMultiErrors()

	if d := needs.Difference(ls); len(d) > 0 {
		mErr.Add(fmt.Sprintf(nonRequiringLabels, strings.Join(d.List(), ",")))
	}

	if i := ls.Intersection(misses); len(i) > 0 {
		mErr.Add(fmt.Sprintf(nonMissingLabels, strings.Join(i.List(), ",")))
	}

	return mErr.Err()
}

func labelsToSets(labels []sdk.Label) sets.String {
	ls := sets.NewString()

	for _, v := range labels {
		ls.Insert(v.Name)
	}

	return ls
}

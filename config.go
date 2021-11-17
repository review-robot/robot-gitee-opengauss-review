package main

import (
	"fmt"

	libconfig "github.com/opensourceways/community-robot-lib/config"
)

type pullRequestMergeType string

const (
	mergeMerge  pullRequestMergeType = "merge"
	mergeSquash pullRequestMergeType = "squash"
)

type configuration struct {
	ConfigItems []botConfig `json:"config_items,omitempty"`
}

func (c *configuration) configFor(org, repo string) *botConfig {
	if c == nil {
		return nil
	}

	items := c.ConfigItems

	v := make([]libconfig.IPluginForRepo, len(items))
	for i := range items {
		v[i] = &items[i]
	}

	if i := libconfig.FindConfig(org, repo, v); i >= 0 {
		return &items[i]
	}
	return nil
}

func (c *configuration) Validate() error {
	if c == nil {
		return nil
	}

	items := c.ConfigItems
	for i := range items {
		if err := items[i].validate(); err != nil {
			return err
		}
	}

	return nil
}

func (c *configuration) SetDefault() {
	if c == nil {
		return
	}

	Items := c.ConfigItems
	for i := range Items {
		Items[i].setDefault()
	}
}

type botConfig struct {
	libconfig.PluginForRepo

	// LgtmCountsRequired specifies the number of lgtm label which will be need for the pr.
	// When it is greater than 1, the lgtm label is composed of 'lgtm-login'.
	// The default value is 1 which means the lgtm label is itself.
	LgtmCountsRequired uint `json:"lgtm_counts_required,omitempty"`

	// ReposOfSig specifies the repos for which it should check the devepler's permission
	// besed on the owners file in sig directory when the developer comment /lgtm or /approve
	// command. The format is 'org/repo'.
	ReposOfSig []string `json:"repos_of_sig,omitempty"`

	// RequiringLabels only PRs that already have these tags can be merged
	RequiringLabels []string `json:"requiring_labels,omitempty"`

	// MissingLabels PRs that already have these tags cannot be merged*
	MissingLabels []string `json:"missing_labels,omitempty"`

	// MergeMethod is the method to merge PR.
	// The default method of merge. Valid options are squash and merge.
	MergeMethod pullRequestMergeType `json:"merge_method,omitempty"`
}

func (c *botConfig) setDefault() {
	if c.LgtmCountsRequired == 0 {
		c.LgtmCountsRequired = 1
	}

	if c.MergeMethod == "" {
		c.MergeMethod = mergeMerge
	}
}

func (c *botConfig) validate() error {
	if c.MergeMethod != mergeMerge && c.MergeMethod != mergeSquash {
		return fmt.Errorf("unsupported merge method:%s", c.MergeMethod)
	}

	return c.PluginForRepo.Validate()
}

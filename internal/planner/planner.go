package planner

import (
	"fmt"
	"path/filepath"

	"github.com/erlcx/cli/internal/config"
	"github.com/erlcx/cli/internal/lockfile"
	"github.com/erlcx/cli/internal/scanner"
)

type Classification string

const (
	ClassUpload         Classification = "upload"
	ClassUnchanged      Classification = "unchanged"
	ClassTemplateMatch  Classification = "template_match"
	ClassConfiguredSkip Classification = "configured_skip"
)

type Plan struct {
	PackDir string
	Items   []Item
	Counts  Counts
}

type Counts struct {
	Total          int
	Upload         int
	Unchanged      int
	TemplateMatch  int
	ConfiguredSkip int
}

type Item struct {
	Image         scanner.ImageFile
	Class         Classification
	Reason        string
	TemplateMatch *scanner.ImageFile
	LockEntry     *lockfile.Entry
}

func BuildScanPlan(packDir string, cfg config.Config, existingLock *lockfile.LockFile) (Plan, error) {
	return BuildScanPlanForCreator(packDir, cfg, lockCreatorFromConfig(cfg), existingLock)
}

func BuildScanPlanForCreator(packDir string, cfg config.Config, targetCreator lockfile.Creator, existingLock *lockfile.LockFile) (Plan, error) {
	images, err := scanner.RequireImages(packDir)
	if err != nil {
		return Plan{}, err
	}

	images, err = scanner.HashImageFiles(images)
	if err != nil {
		return Plan{}, err
	}

	skipMatcher, err := scanner.NewSkipMatcher(cfg.SkipNamePatterns)
	if err != nil {
		return Plan{}, err
	}

	var templateIndex scanner.TemplateIndex
	hasTemplateIndex := cfg.TemplatesDir != ""
	if hasTemplateIndex {
		templatesDir := cfg.TemplatesDir
		if !filepath.IsAbs(templatesDir) {
			templatesDir = filepath.Join(packDir, templatesDir)
		}

		templateIndex, err = scanner.BuildTemplateIndex(templatesDir)
		if err != nil {
			return Plan{}, fmt.Errorf("templates: %w", err)
		}
	}

	plan := Plan{
		PackDir: packDir,
		Items:   make([]Item, 0, len(images)),
	}

	for _, image := range images {
		item := classifyImage(image, cfg, targetCreator, skipMatcher, templateIndex, hasTemplateIndex, existingLock)
		plan.Items = append(plan.Items, item)
		plan.Counts.add(item.Class)
	}
	plan.Counts.Total = len(plan.Items)

	return plan, nil
}

func classifyImage(
	image scanner.ImageFile,
	cfg config.Config,
	targetCreator lockfile.Creator,
	skipMatcher scanner.SkipMatcher,
	templateIndex scanner.TemplateIndex,
	hasTemplateIndex bool,
	existingLock *lockfile.LockFile,
) Item {
	if pattern, ok := skipMatcher.MatchImage(image); ok {
		return Item{
			Image:  image,
			Class:  ClassConfiguredSkip,
			Reason: fmt.Sprintf("matched configured skip pattern %q", pattern),
		}
	}

	if hasTemplateIndex {
		if template, ok := templateIndex.MatchImage(image); ok {
			return Item{
				Image:         image,
				Class:         ClassTemplateMatch,
				Reason:        fmt.Sprintf("matched template %s", template.RelPath),
				TemplateMatch: &template,
			}
		}
	}

	if existingLock != nil && existingLock.CreatorMatches(targetCreator) {
		if entry, ok := existingLock.Files[image.RelPath]; ok && entry.MatchesContent(image.SHA256, cfg.AssetType) {
			return Item{
				Image:     image,
				Class:     ClassUnchanged,
				Reason:    fmt.Sprintf("unchanged decal ID %s", entry.AssetID),
				LockEntry: &entry,
			}
		}
	}

	return Item{
		Image:  image,
		Class:  ClassUpload,
		Reason: "new or changed image",
	}
}

func (counts *Counts) add(class Classification) {
	switch class {
	case ClassUpload:
		counts.Upload++
	case ClassUnchanged:
		counts.Unchanged++
	case ClassTemplateMatch:
		counts.TemplateMatch++
	case ClassConfiguredSkip:
		counts.ConfiguredSkip++
	}
}

func lockCreatorFromConfig(cfg config.Config) lockfile.Creator {
	creator := lockfile.Creator{
		Type: cfg.Creator.Type,
	}
	if cfg.Creator.GroupID != nil {
		creator.ID = fmt.Sprintf("%d", *cfg.Creator.GroupID)
	}
	return creator
}

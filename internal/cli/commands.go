package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/erlcx/cli/internal/auth"
	"github.com/erlcx/cli/internal/config"
	"github.com/erlcx/cli/internal/ids"
	"github.com/erlcx/cli/internal/lockfile"
	"github.com/erlcx/cli/internal/planner"
	"github.com/erlcx/cli/internal/uploader"
)

var newUploaderClient = func() uploader.Client {
	return uploader.Client{
		BaseURL: os.Getenv("ERLCX_ROBLOX_API_BASE_URL"),
	}
}

func runScan(opts fileCommandOptions, output io.Writer) int {
	style := newStyler(output)
	start := time.Now()
	printScanProgress(output, style, opts)
	lock, err := loadOptionalLock(resolvePackPath(opts.PackDir, opts.Config.LockFile))
	if err != nil {
		style.errorf(output, "scan failed: %v", err)
		return 1
	}
	targetCreator, err := resolveCreatorForScan(opts.Config, lock)
	if err != nil {
		style.errorf(output, "scan failed: %v", err)
		return 1
	}

	plan, err := planner.BuildScanPlanForCreator(opts.PackDir, opts.Config, targetCreator, lock)
	if err != nil {
		style.errorf(output, "scan failed: %v", err)
		return 1
	}

	fmt.Fprintf(output, "%s scan completed in %s\n\n", style.green("done:"), time.Since(start).Round(time.Millisecond))
	printScanPlan(output, plan, style, opts.Verbose)
	return 0
}

func runUpload(opts fileCommandOptions, output io.Writer) int {
	style := newStyler(output)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	start := time.Now()

	var token auth.AccessToken
	var targetCreator lockfile.Creator
	if opts.Config.Creator.Type == config.CreatorTypeGroup {
		targetCreator = lockfile.Creator{
			Type: lockfile.CreatorTypeGroup,
			ID:   fmt.Sprintf("%d", *opts.Config.Creator.GroupID),
		}
	} else if !opts.DryRun {
		targetCreator = lockfile.Creator{Type: lockfile.CreatorTypeUser}
	} else {
		lock, _ := loadOptionalLock(resolvePackPath(opts.PackDir, opts.Config.LockFile))
		var err error
		targetCreator, err = resolveCreatorForScan(opts.Config, lock)
		if err != nil {
			style.errorf(output, "upload dry-run failed: %v", err)
			return 1
		}
	}

	if !opts.DryRun {
		var err error
		token, err = newAuthService().AccessToken(ctx, auth.AccessTokenOptions{
			ClientSecret: os.Getenv(authClientSecretEnv),
		})
		if err != nil {
			if errors.Is(err, auth.ErrNotLoggedIn) {
				style.errorf(output, "upload failed: not logged in; run erlcx auth login")
			} else {
				style.errorf(output, "upload failed: %v", err)
			}
			return 1
		}
		if opts.Config.Creator.Type == config.CreatorTypeUser {
			targetCreator = lockfile.Creator{Type: lockfile.CreatorTypeUser, ID: token.Credential.UserID}
		}
	}

	lockPath := resolvePackPath(opts.PackDir, opts.Config.LockFile)
	var lock lockfile.LockFile
	var existingLock *lockfile.LockFile
	var err error
	if opts.DryRun {
		existingLock, err = loadOptionalLock(lockPath)
		if err != nil {
			style.errorf(output, "upload failed: %v", err)
			return 1
		}
		if existingLock != nil {
			lock = *existingLock
		} else {
			lock = lockfile.New(targetCreator)
			existingLock = nil
		}
	} else {
		lock, err = lockfile.LoadOrNew(lockPath, targetCreator)
	}
	if err != nil && !opts.DryRun {
		style.errorf(output, "upload failed: %v", err)
		return 1
	}
	if !opts.DryRun && !lock.CreatorMatches(targetCreator) {
		lock = lockfile.New(targetCreator)
	}

	printScanProgress(output, style, opts)
	if !opts.DryRun {
		existingLock = &lock
	}
	plan, err := planner.BuildScanPlanForCreator(opts.PackDir, opts.Config, targetCreator, existingLock)
	if err != nil {
		style.errorf(output, "upload failed: %v", err)
		return 1
	}

	if opts.DryRun {
		fmt.Fprintf(output, "%s No files will be uploaded.\n\n", style.yellow("dry-run:"))
		fmt.Fprintf(output, "%s scan completed in %s\n\n", style.green("done:"), time.Since(start).Round(time.Millisecond))
		printScanPlan(output, plan, style, opts.Verbose)
		return 0
	}

	fmt.Fprintf(output, "%s scan completed in %s\n\n", style.green("done:"), time.Since(start).Round(time.Millisecond))
	printScanPlan(output, plan, style, opts.Verbose)
	if plan.Counts.Upload == 0 {
		if err := writeIDsFile(resolvePackPath(opts.PackDir, opts.Config.OutputFile), lock); err != nil {
			style.errorf(output, "upload failed: %v", err)
			return 1
		}
		fmt.Fprintf(output, "%s No uploads needed.\n", style.green("ok:"))
		return 0
	}

	jobs := uploadJobsFromPlan(plan, opts.Config, targetCreator)
	now := time.Now().UTC()
	var uploadMu sync.Mutex
	results, uploadErr := newUploaderClient().UploadMany(ctx, token.Token, jobs, uploader.UploadOptions{
		Concurrency: opts.Config.Concurrency,
		FailFast:    opts.FailFast,
		Poll: uploader.PollOptions{
			Interval: 2 * time.Second,
			Timeout:  5 * time.Minute,
		},
		OnResult: func(result uploader.Result) {
			uploadMu.Lock()
			defer uploadMu.Unlock()
			recordUploadResult(output, style, opts, plan, &lock, now, result)
		},
	})
	_ = results

	finalExitCode := 0
	if uploadErr != nil && opts.FailFast {
		style.errorf(output, "upload failed: %v", uploadErr)
		return 1
	}
	if uploadErr != nil {
		fmt.Fprintf(output, "%s upload completed with failures: %v\n", style.yellow("warning:"), uploadErr)
		finalExitCode = 1
	}
	if err := lockfile.Save(lockPath, lock); err != nil {
		style.errorf(output, "upload failed: %v", err)
		return 1
	}
	if err := writeIDsFile(resolvePackPath(opts.PackDir, opts.Config.OutputFile), lock); err != nil {
		style.errorf(output, "upload failed: %v", err)
		return 1
	}

	fmt.Fprintf(output, "%s %s\n", style.green("updated:"), lockPath)
	fmt.Fprintf(output, "%s %s\n", style.green("updated:"), resolvePackPath(opts.PackDir, opts.Config.OutputFile))
	return finalExitCode
}

func recordUploadResult(output io.Writer, style styler, opts fileCommandOptions, plan planner.Plan, lock *lockfile.LockFile, uploadedAt time.Time, result uploader.Result) {
	if result.Job.Request.DisplayName == "" {
		return
	}
	if result.Err != nil {
		if opts.FailFast && errors.Is(result.Err, context.Canceled) {
			return
		}
		fmt.Fprintf(output, "%s %s  %v\n", style.red("x failed"), result.Job.Request.DisplayName, result.Err)
		return
	}

	item := plan.Items[result.Job.Index]
	lock.Files[item.Image.RelPath] = lockfile.Entry{
		SHA256:      item.Image.SHA256,
		AssetType:   lockfile.AssetTypeDecal,
		AssetID:     result.Asset.AssetID,
		DisplayName: item.DisplayName,
		UploadedAt:  uploadedAt,
	}
	fmt.Fprintf(output, "%s %s  %s\n", style.green("+ uploaded"), item.DisplayName, style.cyan(result.Asset.AssetID))
}

func runIDs(opts fileCommandOptions, output io.Writer) int {
	style := newStyler(output)
	lockPath := resolvePackPath(opts.PackDir, opts.Config.LockFile)
	lock, err := lockfile.Load(lockPath)
	if err != nil {
		style.errorf(output, "ids failed: %v", err)
		return 1
	}

	outputPath := resolvePackPath(opts.PackDir, opts.Config.OutputFile)
	if err := writeIDsFile(outputPath, lock); err != nil {
		style.errorf(output, "ids failed: %v", err)
		return 1
	}

	fmt.Fprintf(output, "%s %s\n", style.green("updated:"), outputPath)
	return 0
}

func runLockClean(opts fileCommandOptions, output io.Writer) int {
	style := newStyler(output)
	lockPath := resolvePackPath(opts.PackDir, opts.Config.LockFile)
	lock, err := lockfile.Load(lockPath)
	if err != nil {
		style.errorf(output, "lock clean failed: %v", err)
		return 1
	}

	removed := 0
	for relPath := range lock.Files {
		if _, err := os.Stat(resolvePackPath(opts.PackDir, relPath)); errors.Is(err, os.ErrNotExist) {
			delete(lock.Files, relPath)
			removed++
		} else if err != nil {
			style.errorf(output, "lock clean failed: check %s: %v", relPath, err)
			return 1
		}
	}

	if err := lockfile.Save(lockPath, lock); err != nil {
		style.errorf(output, "lock clean failed: %v", err)
		return 1
	}

	fmt.Fprintf(output, "%s %d stale lock entries\n", style.yellow("removed:"), removed)
	fmt.Fprintf(output, "%s %s\n", style.green("updated:"), lockPath)
	return 0
}

func printScanProgress(output io.Writer, style styler, opts fileCommandOptions) {
	fmt.Fprintf(output, "%s scanning pack %s\n", style.cyan("scan:"), opts.PackDir)
	if opts.Config.TemplatesDir != "" {
		fmt.Fprintf(output, "%s indexing templates %s\n", style.cyan("scan:"), opts.Config.TemplatesDir)
	}
	fmt.Fprintf(output, "%s hashing images; this can take a moment for large packs\n", style.dim("scan:"))
}

func printScanPlan(output io.Writer, plan planner.Plan, style styler, verbose bool) {
	fmt.Fprintf(output, "%s %d images\n", style.bold("Scanned"), plan.Counts.Total)
	fmt.Fprintf(output, "  %s %-18s %d\n", style.green("+"), "upload candidates", plan.Counts.Upload)
	fmt.Fprintf(output, "  %s %-18s %d\n", style.dim("="), "unchanged", plan.Counts.Unchanged)
	fmt.Fprintf(output, "  %s %-18s %d\n", style.yellow("~"), "template matches", plan.Counts.TemplateMatch)
	fmt.Fprintf(output, "  %s %-18s %d\n", style.blue("-"), "configured skips", plan.Counts.ConfiguredSkip)

	items, hidden := visiblePlanItems(plan, verbose)
	if len(items) == 0 {
		if hidden > 0 {
			fmt.Fprintf(output, "\n  %s %d unchanged/template/skip entries hidden; use --verbose to show all\n", style.dim("..."), hidden)
		}
		return
	}
	fmt.Fprintln(output)
	for _, item := range items {
		marker, class := styledClass(style, item.Class)
		fmt.Fprintf(output, "  %s %-15s %s\n", marker, class, item.Image.RelPath)
		fmt.Fprintf(output, "      %s\n", style.dim(item.Reason))
	}
	if hidden > 0 {
		fmt.Fprintf(output, "\n  %s %d unchanged/template/skip entries hidden; use --verbose to show all\n", style.dim("..."), hidden)
	}
}

func visiblePlanItems(plan planner.Plan, verbose bool) ([]planner.Item, int) {
	if verbose {
		return plan.Items, 0
	}

	visible := make([]planner.Item, 0, len(plan.Items))
	hidden := 0
	for _, item := range plan.Items {
		if item.Class == planner.ClassUpload {
			visible = append(visible, item)
		} else {
			hidden++
		}
	}
	return visible, hidden
}

func styledClass(style styler, class planner.Classification) (string, string) {
	switch class {
	case planner.ClassUpload:
		return style.green("+"), style.green(string(class))
	case planner.ClassUnchanged:
		return style.dim("="), style.dim(string(class))
	case planner.ClassTemplateMatch:
		return style.yellow("~"), style.yellow(string(class))
	case planner.ClassConfiguredSkip:
		return style.blue("-"), style.blue(string(class))
	default:
		return "?", string(class)
	}
}

func uploadJobsFromPlan(plan planner.Plan, cfg config.Config, creator lockfile.Creator) []uploader.Job {
	jobs := make([]uploader.Job, 0, plan.Counts.Upload)
	for i, item := range plan.Items {
		if item.Class != planner.ClassUpload {
			continue
		}
		jobs = append(jobs, uploader.Job{
			Index: i,
			Request: uploader.AssetUploadRequest{
				FilePath:    item.Image.AbsPath,
				DisplayName: item.DisplayName,
				AssetType:   cfg.AssetType,
				Creator: uploader.Creator{
					Type: creator.Type,
					ID:   creator.ID,
				},
			},
		})
	}
	return jobs
}

func writeIDsFile(path string, lock lockfile.LockFile) error {
	content, err := ids.Generate(lock)
	if err != nil {
		return fmt.Errorf("generate IDs file: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("write IDs file %s: %w", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return fmt.Errorf("write IDs file %s: %w", path, err)
	}
	return nil
}

func loadOptionalLock(path string) (*lockfile.LockFile, error) {
	lock, err := lockfile.Load(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &lock, nil
}

func resolveCreatorForScan(cfg config.Config, existingLock *lockfile.LockFile) (lockfile.Creator, error) {
	if cfg.Creator.Type == config.CreatorTypeGroup {
		return lockfile.Creator{Type: lockfile.CreatorTypeGroup, ID: fmt.Sprintf("%d", *cfg.Creator.GroupID)}, nil
	}
	status, err := newAuthService().Status(context.Background(), auth.StatusOptions{})
	if err == nil && status.LoggedIn && status.UserID != "" {
		return lockfile.Creator{Type: lockfile.CreatorTypeUser, ID: status.UserID}, nil
	}
	if existingLock != nil && existingLock.Creator.Type == lockfile.CreatorTypeUser {
		return existingLock.Creator, nil
	}
	return lockfile.Creator{Type: lockfile.CreatorTypeUser}, nil
}

func resolvePackPath(packDir string, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(packDir, path)
}

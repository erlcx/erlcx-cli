package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
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
	lock, err := loadOptionalLock(resolvePackPath(opts.PackDir, opts.Config.LockFile))
	if err != nil {
		fmt.Fprintf(output, "scan failed: %v\n", err)
		return 1
	}
	targetCreator, err := resolveCreatorForScan(opts.Config, lock)
	if err != nil {
		fmt.Fprintf(output, "scan failed: %v\n", err)
		return 1
	}

	plan, err := planner.BuildScanPlanForCreator(opts.PackDir, opts.Config, targetCreator, lock)
	if err != nil {
		fmt.Fprintf(output, "scan failed: %v\n", err)
		return 1
	}

	printScanPlan(output, plan)
	return 0
}

func runUpload(opts fileCommandOptions, output io.Writer) int {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

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
			fmt.Fprintf(output, "upload dry-run failed: %v\n", err)
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
				fmt.Fprintln(output, "upload failed: not logged in; run erlcx auth login")
			} else {
				fmt.Fprintf(output, "upload failed: %v\n", err)
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
			fmt.Fprintf(output, "upload failed: %v\n", err)
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
		fmt.Fprintf(output, "upload failed: %v\n", err)
		return 1
	}
	if !opts.DryRun && !lock.CreatorMatches(targetCreator) {
		lock = lockfile.New(targetCreator)
	}

	if !opts.DryRun {
		existingLock = &lock
	}
	plan, err := planner.BuildScanPlanForCreator(opts.PackDir, opts.Config, targetCreator, existingLock)
	if err != nil {
		fmt.Fprintf(output, "upload failed: %v\n", err)
		return 1
	}

	if opts.DryRun {
		fmt.Fprintln(output, "Dry run. No files will be uploaded.")
		printScanPlan(output, plan)
		return 0
	}

	printScanPlan(output, plan)
	if plan.Counts.Upload == 0 {
		if err := writeIDsFile(resolvePackPath(opts.PackDir, opts.Config.OutputFile), lock); err != nil {
			fmt.Fprintf(output, "upload failed: %v\n", err)
			return 1
		}
		fmt.Fprintln(output, "No uploads needed.")
		return 0
	}

	jobs := uploadJobsFromPlan(plan, opts.Config, targetCreator)
	results, uploadErr := newUploaderClient().UploadMany(ctx, token.Token, jobs, uploader.UploadOptions{
		Concurrency: opts.Config.Concurrency,
		FailFast:    opts.FailFast,
		Poll: uploader.PollOptions{
			Interval: 2 * time.Second,
			Timeout:  5 * time.Minute,
		},
	})

	now := time.Now().UTC()
	for _, result := range results {
		if result.Job.Request.DisplayName == "" {
			continue
		}
		if result.Err != nil {
			fmt.Fprintf(output, "Failed %s: %v\n", result.Job.Request.DisplayName, result.Err)
			continue
		}
		item := plan.Items[result.Job.Index]
		lock.Files[item.Image.RelPath] = lockfile.Entry{
			SHA256:      item.Image.SHA256,
			AssetType:   lockfile.AssetTypeDecal,
			AssetID:     result.Asset.AssetID,
			DisplayName: item.DisplayName,
			UploadedAt:  now,
		}
		fmt.Fprintf(output, "Uploaded %s: %s\n", item.DisplayName, result.Asset.AssetID)
	}

	if uploadErr != nil {
		fmt.Fprintf(output, "upload failed: %v\n", uploadErr)
		return 1
	}
	if err := lockfile.Save(lockPath, lock); err != nil {
		fmt.Fprintf(output, "upload failed: %v\n", err)
		return 1
	}
	if err := writeIDsFile(resolvePackPath(opts.PackDir, opts.Config.OutputFile), lock); err != nil {
		fmt.Fprintf(output, "upload failed: %v\n", err)
		return 1
	}

	fmt.Fprintf(output, "Updated %s\n", lockPath)
	fmt.Fprintf(output, "Updated %s\n", resolvePackPath(opts.PackDir, opts.Config.OutputFile))
	return 0
}

func printScanPlan(output io.Writer, plan planner.Plan) {
	fmt.Fprintf(output, "Scanned %d images\n", plan.Counts.Total)
	fmt.Fprintf(output, "Upload candidates: %d\n", plan.Counts.Upload)
	fmt.Fprintf(output, "Unchanged: %d\n", plan.Counts.Unchanged)
	fmt.Fprintf(output, "Template matches: %d\n", plan.Counts.TemplateMatch)
	fmt.Fprintf(output, "Configured skips: %d\n", plan.Counts.ConfiguredSkip)

	for _, item := range plan.Items {
		fmt.Fprintf(output, "%s  %s  %s\n", item.Class, item.Image.RelPath, item.Reason)
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

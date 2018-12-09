package cmd

import (
	"fmt"
	"log"

	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"syscall"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/blang/semver"
	"github.com/pkg/errors"
	"github.com/robfig/cron"
)

// Execute will actually do the thing
func Execute(cfg *CleanerConfig, dryRun bool) error {
	sigs := make(chan os.Signal)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	if dryRun {
		return DryRun(cfg)
	}

	c := cron.New()
	c.AddFunc(cfg.Schedule, CleanupFiles(cfg))
	c.Start()

	for {
		sig := <-sigs
		switch sig {
		case syscall.SIGHUP:
			entries := c.Entries()
			if len(entries) > 0 {
				e := entries[0]
				fmt.Fprintln(cfg.Writer(), "Cleanup will occur next at", e.Next.Format(time.RFC3339))
			}
		default:
			return nil
		}
	}
}

// ParseConfig will attempt to read the config toml
func ParseConfig(configFile string) (*CleanerConfig, error) {
	log.Println("Reading config file", configFile)
	conf := new(CleanerConfig)
	if _, err := toml.DecodeFile(configFile, conf); err != nil {
		return nil, err
	}

	conf.SetWriter(os.Stdout)

	return conf, nil
}

// CleanupFiles will delete the files that are older than we want to keep
func CleanupFiles(cfg *CleanerConfig) func() {
	return func() {
		paths, err := getFilesToDelete(cfg)
		if err != nil {
			log.Println(err)
			return
		}

		for _, path := range paths {
			fmt.Fprintln(cfg.Writer(), "Deleting", path)
			err := os.Remove(path)
			if err != nil {
				log.Println(err)
				return
			}
		}
	}
}

// DryRun will list the files that would be deleted by CleanupFiles
func DryRun(cfg *CleanerConfig) error {
	paths, err := getFilesToDelete(cfg)
	if err != nil {
		return err
	}

	for _, path := range paths {
		fmt.Fprintln(cfg.Writer(), "Will delete:", path[len(cfg.BaseDirectory)+1:])
	}

	return nil
}

func getFilesToDelete(cfg *CleanerConfig) ([]string, error) {
	allMatchingFiles := make(map[File][][2]string)
	filesToDelete := make([]string, 0)
	for _, bucket := range cfg.Buckets {
		baseDir := filepath.Join(cfg.BaseDirectory, bucket.Name)

		for _, filePattern := range bucket.Files {
			pat := regexp.MustCompile(filePattern.Pattern)
			files := make([][2]string, 0)
			err := filepath.Walk(baseDir, walkDir(cfg.BaseDirectory, pat, &files))
			if err != nil {
				return nil, errors.Wrap(err, "Error walking bucket")
			}
			allMatchingFiles[filePattern] = files
		}
	}

	for fileType, matchingFiles := range allMatchingFiles {
		versions := make([]semver.Version, len(matchingFiles))
		versionMap := make(map[string]string)
		for i, mf := range matchingFiles {
			v, err := semver.ParseTolerant(mf[0])
			if err != nil {
				return nil, err
			}
			versions[i] = v
			versionMap[v.String()] = mf[1]
		}

		semver.Sort(versions)

		var versionsToDelete []semver.Version
		if len(versions) <= int(fileType.Retainer) {
			versionsToDelete = []semver.Version{}
		} else {
			versionsToDelete = versions[:len(versions)-int(fileType.Retainer)]
		}

		for _, v := range versionsToDelete {
			filesToDelete = append(filesToDelete, versionMap[v.String()])
		}
	}

	return filesToDelete, nil
}

func walkDir(baseDir string, pattern *regexp.Regexp, files *[][2]string) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("prevent panic by handling failure accessing a path %q: %v\n", path, err)
			return err
		}

		baseName := filepath.Base(path)
		dirName := filepath.Dir(path)

		if info.IsDir() && dirName != baseDir {
			return filepath.SkipDir
		}

		matches := pattern.FindStringSubmatch(baseName)
		if len(matches) == 2 {
			*files = append(*files, [2]string{matches[1], path})
		}
		return nil
	}
}

package cmd_test

import (
	"bytes"
	"cleaner/cmd"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestCleanerCommand(t *testing.T) {
	spec.Run(t, "CleanerCommand", testCleanerCommand, spec.Report(report.Terminal{}))
}

func testCleanerCommand(t *testing.T, when spec.G, it spec.S) {
	it.Before(func() {
		RegisterTestingT(t)
	})

	when("A config file needs parsed", func() {
		it("can be", func() {
			config, err := cmd.ParseConfig("testdata/config.toml")
			Expect(err).NotTo(HaveOccurred())
			Expect(config.BaseDirectory).To(Equal("testdata/base-dir"))
			Expect(config.Schedule).To(Equal("@weekly"))
			Expect(config.Buckets).To(HaveLen(2))
		})
	})

	when("running cleaner", func() {
		var config *cmd.CleanerConfig
		it.Before(func() {
			config = &cmd.CleanerConfig{
				BaseDirectory: "testdata/base-dir",
				Schedule:      "@every 2s",
				Buckets: []cmd.Bucket{
					cmd.Bucket{
						Name: "a",
						Files: []cmd.File{
							cmd.File{
								Pattern:  "tile1-(.*).pivotal",
								Retainer: 3,
							},
							cmd.File{
								Pattern:  "tile2-(.*).pivotal",
								Retainer: 4,
							},
						},
					},
					cmd.Bucket{
						Name: "b",
						Files: []cmd.File{
							cmd.File{
								Pattern:  "sc-(.*).tgz",
								Retainer: 2,
							},
						},
					},
					cmd.Bucket{
						Name: "c",
						Files: []cmd.File{
							cmd.File{
								Pattern:  "foo-(.*).ova",
								Retainer: 0,
							},
						},
					},
					cmd.Bucket{
						Name: "busted",
						Files: []cmd.File{
							cmd.File{
								Pattern:  "weird-(.*).txt",
								Retainer: 1,
							},
						},
					},
				},
			}

			preseedFiles()
		})

		when("Doing a dry run", func() {
			it("lists files to exist but does not delete them", func() {
				b := bytes.NewBuffer([]byte{})
				config.SetWriter(b)

				err := cmd.DryRun(config)
				Expect(err).NotTo(HaveOccurred())

				outputStr := b.String()
				lines := strings.Split(strings.TrimSpace(outputStr), "\n")
				Expect(lines).To(HaveLen(9))
				Expect(lines).To(ContainElement("Will delete: a/tile1-1.0.0-beta2.pivotal"))
				Expect(lines).To(ContainElement("Will delete: a/tile1-1.0.0-beta1.pivotal"))
				Expect(lines).To(ContainElement("Will delete: a/tile2-2.3.4.pivotal"))
				Expect(lines).To(ContainElement("Will delete: a/tile2-2.3.5.pivotal"))
				Expect(lines).To(ContainElement("Will delete: a/tile2-2.3.6.pivotal"))
				Expect(lines).To(ContainElement("Will delete: c/foo-1.23.4.ova"))
				Expect(lines).To(ContainElement("Will delete: busted/weird-1.2.3.txt"))
				Expect(lines).To(ContainElement("Will delete: busted/weird-1.2.3-beta1.txt"))
				Expect(lines).To(ContainElement("Will delete: busted/weird-2.4-build3.txt"))

			})
		})

		when("Doing a real run", func() {
			it("works", func() {
				config.SetWriter(ioutil.Discard)
				cmd.CleanupFiles(config)()
				Expect("testdata/base-dir/a/tile1-1.0.0-beta2.pivotal").NotTo(BeAnExistingFile())
				Expect("testdata/base-dir/a/tile1-1.0.0-beta1.pivotal").NotTo(BeAnExistingFile())
				Expect("testdata/base-dir/a/tile2-2.3.4.pivotal").NotTo(BeAnExistingFile())
				Expect("testdata/base-dir/a/tile2-2.3.5.pivotal").NotTo(BeAnExistingFile())
				Expect("testdata/base-dir/a/tile2-2.3.6.pivotal").NotTo(BeAnExistingFile())
				Expect("testdata/base-dir/c/foo-1.23.4.ova").NotTo(BeAnExistingFile())
			})
		})

		when("Sending the HUP signal", func() {
			it("lets us know the next time the job will run", func() {
				b := &bytes.Buffer{}
				config.SetWriter(b)
				config.Schedule = "@weekly"

				go cmd.Execute(config, false)

				// Give the process a chance to start before we start sending it signals
				time.Sleep(50 * time.Millisecond)

				p, err := os.FindProcess(os.Getpid())
				Expect(err).NotTo(HaveOccurred())

				err = p.Signal(syscall.SIGHUP)
				Expect(err).NotTo(HaveOccurred())

				now := time.Now()

				// Give the process a chance to write to the buffer before we check it
				time.Sleep(50 * time.Millisecond)

				var timestamp string
				_, err = fmt.Sscanf(b.String(), "Cleanup will occur next at %s\n", &timestamp)
				Expect(err).NotTo(HaveOccurred())

				next, err := time.Parse(time.RFC3339, timestamp)
				Expect(err).NotTo(HaveOccurred())

				duration := next.Sub(now)
				Expect(duration > 0).To(BeTrue())
				Expect(duration < 168*time.Hour).To(BeTrue())
			})
		})
	})
}

func preseedFiles() {
	c := func(f string) {
		p, e := os.Create(filepath.Join("testdata/base-dir", f))
		if !os.IsExist(e) {
			Expect(e).NotTo(HaveOccurred())
		}
		defer p.Close()
	}

	c("a/tile1-1.0.0-beta1.pivotal")
	c("a/tile1-1.0.0-beta2.pivotal")
	c("a/tile1-1.0.0-rc1.pivotal")
	c("a/tile1-1.0.0.pivotal")
	c("a/tile1-1.0.2.pivotal")
	c("a/tile2-2.3.10.pivotal")
	c("a/tile2-2.3.4.pivotal")
	c("a/tile2-2.3.5.pivotal")
	c("a/tile2-2.3.6.pivotal")
	c("a/tile2-2.3.7.pivotal")
	c("a/tile2-2.3.8.pivotal")
	c("a/tile2-2.3.9.pivotal")
	c("b/sc-3498.6.tgz")
	c("b/sc-3498.5.tgz")
	c("c/foo-1.23.4.ova")
	c("c/foo.ova")
	c("busted/weird-1.2.3.txt")
	c("busted/weird-1.2.3-beta1.txt")
	c("busted/weird-2.4-build3.txt")
	c("busted/weird-3-rc1.txt")

}

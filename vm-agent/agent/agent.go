package agent

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/mdlayher/vsock"
	"google.golang.org/protobuf/proto"

	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
)

// FakeAgent generates and sends index reports over vsock
type FakeAgent struct {
	port uint32
}

// NewFakeAgent creates a new fake agent
func NewFakeAgent(port uint32) *FakeAgent {
	return &FakeAgent{
		port: port,
	}
}

// Run starts the fake agent that generates and sends index reports
func (a *FakeAgent) Run(ctx context.Context) error {
	// Connect to the host via vsock
	conn, err := vsock.Dial(vsock.Host, a.port, nil)
	if err != nil {
		return fmt.Errorf("failed to dial vsock: %v", err)
	}
	defer conn.Close()

	log.Printf("Connected to host via vsock on port %d", a.port)

	// Generate and send reports periodically
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Fake agent stopping")
			return nil
		case <-ticker.C:
			report := a.generateIndexReport()
			if err := a.sendReport(conn, report); err != nil {
				log.Printf("Failed to send report: %v", err)
				continue
			}
			log.Printf("Sent index report with hash_id: %s", report.HashId)
		}
	}
}

// generateIndexReport creates a realistic v4.IndexReport with various packages
func (a *FakeAgent) generateIndexReport() *v4.IndexReport {
	// Generate a unique hash ID
	hashID := fmt.Sprintf("vm-report-%d", time.Now().Unix())

	// Create realistic package data
	packages := a.generatePackages()
	distributions := a.generateDistributions()
	repositories := a.generateRepositories()

	return &v4.IndexReport{
		HashId:  hashID,
		State:   "IndexFinished",
		Success: true,
		Err:     "",
		Contents: &v4.Contents{
			Packages:      packages,
			Distributions: distributions,
			Repositories:  repositories,
		},
	}
}

// generatePackages creates a variety of realistic packages
func (a *FakeAgent) generatePackages() []*v4.Package {
	packageTemplates := []struct {
		name    string
		version string
		kind    string
		arch    string
		pkgDb   string
	}{
		{"openssl", "1.1.1f-1ubuntu2.16", "binary", "amd64", "sqlite:usr/share/rpm"},
		{"curl", "7.68.0-1ubuntu2.7", "binary", "amd64", "sqlite:usr/share/rpm"},
		{"systemd", "245.4-4ubuntu3.15", "binary", "amd64", "sqlite:usr/share/rpm"},
		{"bash", "5.0-6ubuntu1.2", "binary", "amd64", "sqlite:usr/share/rpm"},
		{"coreutils", "8.30-3ubuntu2", "binary", "amd64", "sqlite:usr/share/rpm"},
		{"libc6", "2.31-0ubuntu9.9", "binary", "amd64", "sqlite:usr/share/rpm"},
		{"python3", "3.8.10-0ubuntu1~20.04.5", "binary", "amd64", "sqlite:usr/share/rpm"},
		{"docker-ce", "5:20.10.21~3-0~ubuntu-focal", "binary", "amd64", "sqlite:usr/share/rpm"},
		{"kubectl", "1.25.4-00", "binary", "amd64", "sqlite:usr/share/rpm"},
		{"nginx", "1.18.0-0ubuntu1.4", "binary", "amd64", "sqlite:usr/share/rpm"},
	}

	var packages []*v4.Package
	for i, template := range packageTemplates {
		// Add some variation to versions occasionally
		version := template.version
		if rand.Float32() < 0.1 { // 10% chance to vary version
			version = fmt.Sprintf("%s-modified", template.version)
		}

		packages = append(packages, &v4.Package{
			Id:        fmt.Sprintf("pkg-%d", i+1),
			Name:      template.name,
			Version:   version,
			Kind:      template.kind,
			Arch:      template.arch,
			PackageDb: template.pkgDb,
		})
	}

	return packages
}

// generateDistributions creates distribution information
func (a *FakeAgent) generateDistributions() []*v4.Distribution {
	distributions := []string{"ubuntu-20.04", "ubuntu-22.04", "rhel-8", "rhel-9"}
	selected := distributions[rand.Intn(len(distributions))]

	switch selected {
	case "ubuntu-20.04":
		return []*v4.Distribution{
			{
				Id:         "ubuntu-20.04",
				Did:        "ubuntu",
				Name:       "Ubuntu",
				Version:    "20.04",
				VersionId:  "20.04",
				Arch:       "amd64",
				PrettyName: "Ubuntu 20.04.3 LTS",
			},
		}
	case "ubuntu-22.04":
		return []*v4.Distribution{
			{
				Id:         "ubuntu-22.04",
				Did:        "ubuntu",
				Name:       "Ubuntu",
				Version:    "22.04",
				VersionId:  "22.04",
				Arch:       "amd64",
				PrettyName: "Ubuntu 22.04.1 LTS",
			},
		}
	case "rhel-8":
		return []*v4.Distribution{
			{
				Id:         "rhel-8",
				Did:        "rhel",
				Name:       "Red Hat Enterprise Linux",
				Version:    "8.7",
				VersionId:  "8.7",
				Arch:       "x86_64",
				PrettyName: "Red Hat Enterprise Linux 8.7 (Ootpa)",
			},
		}
	case "rhel-9":
		return []*v4.Distribution{
			{
				Id:         "rhel-9",
				Did:        "rhel",
				Name:       "Red Hat Enterprise Linux",
				Version:    "9.1",
				VersionId:  "9.1",
				Arch:       "x86_64",
				PrettyName: "Red Hat Enterprise Linux 9.1 (Plow)",
			},
		}
	}

	// Default fallback
	return []*v4.Distribution{
		{
			Id:         "ubuntu-20.04",
			Did:        "ubuntu",
			Name:       "Ubuntu",
			Version:    "20.04",
			VersionId:  "20.04",
			Arch:       "amd64",
			PrettyName: "Ubuntu 20.04.3 LTS",
		},
	}
}

// generateRepositories creates repository information
func (a *FakeAgent) generateRepositories() []*v4.Repository {
	repositories := []string{"ubuntu-main", "ubuntu-security", "rhel-baseos", "rhel-appstream"}
	selected := repositories[rand.Intn(len(repositories))]

	switch selected {
	case "ubuntu-main":
		return []*v4.Repository{
			{
				Id:   "ubuntu-main",
				Name: "Ubuntu Main",
				Uri:  "http://archive.ubuntu.com/ubuntu/",
			},
		}
	case "ubuntu-security":
		return []*v4.Repository{
			{
				Id:   "ubuntu-security",
				Name: "Ubuntu Security",
				Uri:  "http://security.ubuntu.com/ubuntu/",
			},
		}
	case "rhel-baseos":
		return []*v4.Repository{
			{
				Id:   "rhel-baseos",
				Name: "Red Hat Enterprise Linux BaseOS",
				Uri:  "https://cdn.redhat.com/content/dist/rhel8/8/x86_64/baseos/os",
			},
		}
	case "rhel-appstream":
		return []*v4.Repository{
			{
				Id:   "rhel-appstream",
				Name: "Red Hat Enterprise Linux AppStream",
				Uri:  "https://cdn.redhat.com/content/dist/rhel8/8/x86_64/appstream/os",
			},
		}
	}

	// Default fallback
	return []*v4.Repository{
		{
			Id:   "ubuntu-main",
			Name: "Ubuntu Main",
			Uri:  "http://archive.ubuntu.com/ubuntu/",
		},
	}
}

// sendReport serializes and sends the index report over vsock
func (a *FakeAgent) sendReport(conn *vsock.Conn, report *v4.IndexReport) error {
	// Serialize the report to protobuf
	data, err := proto.Marshal(report)
	if err != nil {
		return fmt.Errorf("failed to marshal index report: %v", err)
	}

	// Send the data over vsock
	_, err = conn.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write to vsock connection: %v", err)
	}

	return nil
}

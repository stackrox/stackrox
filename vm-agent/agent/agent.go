package agent

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/mdlayher/vsock"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	vmv1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"google.golang.org/protobuf/proto"
)

// FakeAgent generates and sends index reports over vsock
type FakeAgent struct {
	port         uint32
	packageCount int
}

// NewFakeAgent creates a new fake agent
func NewFakeAgent(port uint32, packageCount int) *FakeAgent {
	return &FakeAgent{
		port:         port,
		packageCount: packageCount,
	}
}

// Run starts the fake agent that generates and sends index reports
func (a *FakeAgent) Run(ctx context.Context) error {
	log.Printf("Fake agent starting - will send reports every 10 seconds on port %d", a.port)

	// Send first report immediately, then every 10 seconds
	for {
		// Check for shutdown before each iteration
		select {
		case <-ctx.Done():
			log.Println("Fake agent stopping")
			return nil
		default:
		}

		log.Printf("🔄 Starting to generate and send index report...")
		report := a.generateIndexReport()
		log.Printf("📊 Generated report with vsock_cid: %s, hash_id: %s, packages: %d, distributions: %d, repositories: %d",
			report.VsockCid, report.IndexV4.HashId, len(report.IndexV4.Contents.Packages), len(report.IndexV4.Contents.Distributions), len(report.IndexV4.Contents.Repositories))

		// Connect, send, and close for each report
		if err := a.connectAndSendReport(report); err != nil {
			log.Printf("❌ Failed to send report: %v", err)
		} else {
			log.Printf("✅ Successfully sent index report with vsock_cid: %s, hash_id: %s", report.VsockCid, report.IndexV4.HashId)
		}

		// Wait 10 seconds before next report
		select {
		case <-ctx.Done():
			log.Println("Fake agent stopping")
			return nil
		case <-time.After(10 * time.Second):
			// Continue to next iteration
		}
	}
}

// connectAndSendReport opens a new vsock connection, sends the report, and closes the connection
func (a *FakeAgent) connectAndSendReport(report *vmv1.IndexReport) error {
	log.Printf("🔌 Opening new vsock connection to host on port %d", a.port)

	// Connect to the host via vsock
	conn, err := vsock.Dial(vsock.Host, a.port, nil)
	if err != nil {
		return fmt.Errorf("failed to dial vsock: %v", err)
	}
	defer func() {
		log.Printf("🔌 Closing vsock connection")
		conn.Close()
	}()

	log.Printf("✅ Successfully connected to host via vsock on port %d", a.port)

	// Send the report
	if err := a.sendReport(conn, report); err != nil {
		return fmt.Errorf("failed to send report: %v", err)
	}

	return nil
}

// generateIndexReport creates a realistic virtualmachine.v1.IndexReport with various packages
func (a *FakeAgent) generateIndexReport() *vmv1.IndexReport {
	// Generate a unique hash ID
	hashID := fmt.Sprintf("vm-report-%d", time.Now().Unix())

	vsockCID, err := vsock.ContextID()
	if err != nil {
		log.Printf("Failed to get vsockCID")
		vsockCID = 42
	}

	// Create realistic package data
	packages := a.generatePackages()
	distributions := a.generateDistributions()
	repositories := a.generateRepositories()

	// Create the v4 index report
	v4Report := &v4.IndexReport{
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

	// Wrap it in a virtualmachine.v1.IndexReport
	return &vmv1.IndexReport{
		VsockCid: fmt.Sprintf("%d", vsockCID),
		IndexV4:  v4Report,
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
		{"git", "1:2.25.1-1ubuntu3.11", "binary", "amd64", "sqlite:usr/share/rpm"},
		{"vim", "2:8.1.2269-1ubuntu5.12", "binary", "amd64", "sqlite:usr/share/rpm"},
		{"wget", "1.20.3-1ubuntu2", "binary", "amd64", "sqlite:usr/share/rpm"},
		{"htop", "2.2.0-2build1", "binary", "amd64", "sqlite:usr/share/rpm"},
		{"tree", "1.8.0-1", "binary", "amd64", "sqlite:usr/share/rpm"},
		{"jq", "1.6-1ubuntu0.20.04.1", "binary", "amd64", "sqlite:usr/share/rpm"},
		{"unzip", "6.0-25ubuntu1.1", "binary", "amd64", "sqlite:usr/share/rpm"},
		{"zip", "3.0-11build1", "binary", "amd64", "sqlite:usr/share/rpm"},
		{"rsync", "3.1.3-8ubuntu0.5", "binary", "amd64", "sqlite:usr/share/rpm"},
		{"tcpdump", "4.9.3-4ubuntu0.2", "binary", "amd64", "sqlite:usr/share/rpm"},
		{"netcat-openbsd", "1.206-1ubuntu1", "binary", "amd64", "sqlite:usr/share/rpm"},
		{"telnet", "0.17-41.2build1", "binary", "amd64", "sqlite:usr/share/rpm"},
		{"ssh", "1:8.2p1-4ubuntu0.8", "binary", "amd64", "sqlite:usr/share/rpm"},
		{"sudo", "1.8.31-1ubuntu1.5", "binary", "amd64", "sqlite:usr/share/rpm"},
		{"cron", "3.0pl1-136ubuntu1.3", "binary", "amd64", "sqlite:usr/share/rpm"},
		{"logrotate", "3.14.0-4ubuntu3.1", "binary", "amd64", "sqlite:usr/share/rpm"},
		{"rsyslog", "8.2001.0-1ubuntu1.4", "binary", "amd64", "sqlite:usr/share/rpm"},
		{"iptables", "1.8.4-3ubuntu2.1", "binary", "amd64", "sqlite:usr/share/rpm"},
		{"ufw", "0.36-6ubuntu1", "binary", "amd64", "sqlite:usr/share/rpm"},
		{"fail2ban", "0.11.1-1", "binary", "amd64", "sqlite:usr/share/rpm"},
	}

	// Limit the number of packages based on the configured count
	packageCount := a.packageCount
	if packageCount > len(packageTemplates) {
		packageCount = len(packageTemplates)
	}
	if packageCount < 1 {
		packageCount = 1
	}

	var packages []*v4.Package
	for i := 0; i < packageCount; i++ {
		template := packageTemplates[i]

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
func (a *FakeAgent) sendReport(conn *vsock.Conn, report *vmv1.IndexReport) error {
	log.Printf("📤 Serializing index report to protobuf...")

	// Serialize the report to protobuf
	data, err := proto.Marshal(report)
	if err != nil {
		return fmt.Errorf("failed to marshal index report: %v", err)
	}

	log.Printf("📦 Serialized report size: %d bytes", len(data))
	log.Printf("📡 Sending data over vsock connection...")

	// Send the data over vsock
	bytesWritten, err := conn.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write to vsock connection: %v", err)
	}

	log.Printf("✅ Successfully wrote %d bytes to vsock connection", bytesWritten)
	return nil
}

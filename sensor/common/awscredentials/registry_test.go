package awscredentials

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/docker/config"
)

func Test_ecrCredentialsManager_GetDockerConfigEntry(t *testing.T) {
	type fields struct {
		dockerConfigEntry *config.DockerConfigEntry
		ecrClient         *ecr.ECR
		expiresAt         time.Time
		stopSignal        concurrency.Signal
	}
	type args struct {
		registry string
	}
	type test struct {
		name   string
		fields fields
		args   args
		want   *RegistryCredentials
	}
	sampleConfig := config.DockerConfigEntry{Username: "foo", Password: "bar"}
	now := time.Now()
	tests := []test{
		{
			name:   "should return nil if token is invalid.",
			fields: fields{},
			args:   args{"123.dkr.ecr.foo-bar-1.amazonaws.com"},
			want:   nil,
		},
		{
			name: "should return nil if not ECR registry.",
			fields: fields{
				dockerConfigEntry: &sampleConfig,
				// Expires in the future.
				expiresAt: time.Now().Add(time.Hour),
			},
			args: args{"docker.io"},
			want: nil,
		},
		{
			name: "should return docker config if token valid.",
			fields: fields{
				dockerConfigEntry: &sampleConfig,
				// Expires in the future.
				expiresAt: now.Add(time.Hour),
			},
			args: args{"123.dkr.ecr.foo-bar-1.amazonaws.com"},
			want: &RegistryCredentials{
				AWSAccount:   "123",
				AWSRegion:    "foo-bar-1",
				DockerConfig: &sampleConfig,
				ExpirestAt:   now.Add(time.Hour),
			},
		},
	}
	for i, r := range []string{
		"dkr.ecr.foo-bar-1.amazonaws.com",        // missing account
		"1234.dkr.ecr.amazonaws.com",             // missing region
		"foobar.dkr.ecr.foo-bar-1.amazonaws.com", // invalid account
	} {
		tests = append(tests, test{
			name: fmt.Sprintf("should return docker config regex is valid#%d.", i),
			fields: fields{
				dockerConfigEntry: &sampleConfig,
				// Expires in the future.
				expiresAt: time.Now().Add(time.Hour),
			},
			args: args{registry: r},
			want: nil,
		})
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &ecrCredentialsManager{
				dockerConfigEntry: tt.fields.dockerConfigEntry,
				ecrClient:         tt.fields.ecrClient,
				expiresAt:         tt.fields.expiresAt,
				stopSignal:        tt.fields.stopSignal,
			}
			if got := m.GetRegistryCredentials(tt.args.registry); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetRegistryCredentials() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFindECRURLAccountAndRegion(t *testing.T) {
	type args struct {
		registry string
	}
	tests := []struct {
		name        string
		args        args
		wantAccount string
		wantRegion  string
		wantOk      bool
	}{
		{
			name:        "Valid ECR URL",
			args:        args{"1234.dkr.ecr.foo-bar.amazonaws.com"},
			wantAccount: "1234",
			wantRegion:  "foo-bar",
			wantOk:      true,
		},
		{
			name:   "Invalid ECR URL, missing account",
			args:   args{"dkr.ecr.foo-bar.amazonaws.com"},
			wantOk: false,
		},
		{
			name:   "Invalid ECR URL, missing region",
			args:   args{"1234.dkr.ecr.amazonaws.com"},
			wantOk: false,
		},
		{
			name:   "Invalid ECR URL, bad account",
			args:   args{"foobar.dkr.ecr.foo-bar-1.amazonaws.com"},
			wantOk: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotAccount, gotRegion, gotOk := findECRURLAccountAndRegion(tt.args.registry)
			if gotAccount != tt.wantAccount {
				t.Errorf("findECRURLAccountAndRegion() gotAccount = %v, want %v", gotAccount, tt.wantAccount)
			}
			if gotRegion != tt.wantRegion {
				t.Errorf("findECRURLAccountAndRegion() gotRegion = %v, want %v", gotRegion, tt.wantRegion)
			}
			if gotOk != tt.wantOk {
				t.Errorf("findECRURLAccountAndRegion() gotOk = %v, want %v", gotOk, tt.wantOk)
			}
		})
	}
}

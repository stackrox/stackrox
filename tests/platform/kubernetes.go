package platform

type kubernetesPlatform struct {
}

func (*kubernetesPlatform) SetupScript() string {
	return `scripts/kubernetes/setup.sh`
}

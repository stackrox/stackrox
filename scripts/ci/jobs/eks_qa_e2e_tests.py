#!/usr/bin/env -S python3 -u

"""
Run qa-tests-backend in an EKS cluster provided via automation-flavors/eks.
"""
import json
import os
import subprocess
import time
from base_qa_e2e_test import make_qa_e2e_test_runner
from clusters import AutomationFlavorsCluster


def install_ebs_csi_driver():
    """Install AWS EBS CSI driver addon for EKS to enable PVC support.

    Scanner V4 DB requires PVCs, but EKS clusters don't have the EBS CSI driver
    by default. The gp2 StorageClass exists but its provisioner (kubernetes.io/aws-ebs)
    is redirected to ebs.csi.aws.com via CSI migration, which isn't running without
    this addon.
    """
    print("Installing AWS EBS CSI driver addon...")

    # Get cluster name from kubectl context or CLUSTER_NAME env var
    cluster_name = os.environ.get("CLUSTER_NAME")
    if not cluster_name:
        result = subprocess.run(
            ["kubectl", "config", "current-context"],
            capture_output=True,
            text=True,
            check=True,
        )
        context = result.stdout.strip()
        # EKS contexts look like: arn:aws:eks:region:account:cluster/cluster-name
        # or sometimes just cluster-name
        if "/cluster/" in context:
            cluster_name = context.split("/cluster/")[-1]
        else:
            cluster_name = context

    print(f"Cluster name: {cluster_name}")

    # Get AWS region
    region = os.environ.get("AWS_DEFAULT_REGION", "us-west-2")
    print(f"AWS region: {region}")

    # Get node IAM role and attach EBS CSI policy
    # First, get a node's IAM role
    print("Getting node IAM role...")
    result = subprocess.run(
        ["kubectl", "get", "nodes", "-o", "json"],
        capture_output=True,
        text=True,
        check=True,
    )
    nodes = json.loads(result.stdout)
    if not nodes.get("items"):
        raise RuntimeError("No nodes found in cluster")

    # Extract instance ID from first node's provider ID
    # Provider ID format: aws:///zone/instance-id
    provider_id = nodes["items"][0]["spec"]["providerID"]
    instance_id = provider_id.split("/")[-1]
    print(f"Node instance ID: {instance_id}")

    # Get IAM instance profile
    result = subprocess.run(
        ["aws", "ec2", "describe-instances",
         "--instance-ids", instance_id,
         "--region", region,
         "--query", "Reservations[0].Instances[0].IamInstanceProfile.Arn",
         "--output", "text"],
        capture_output=True,
        text=True,
        check=True,
    )
    instance_profile_arn = result.stdout.strip()
    instance_profile_name = instance_profile_arn.split("/")[-1]
    print(f"Instance profile: {instance_profile_name}")

    # Get role name from instance profile
    result = subprocess.run(
        ["aws", "iam", "get-instance-profile",
         "--instance-profile-name", instance_profile_name,
         "--query", "InstanceProfile.Roles[0].RoleName",
         "--output", "text"],
        capture_output=True,
        text=True,
        check=True,
    )
    role_name = result.stdout.strip()
    print(f"IAM role: {role_name}")

    # Attach EBS CSI policy to role
    print("Attaching EBS CSI policy to IAM role...")
    subprocess.run(
        ["aws", "iam", "attach-role-policy",
         "--role-name", role_name,
         "--policy-arn", "arn:aws:iam::aws:policy/service-role/AmazonEBSCSIDriverPolicy"],
        check=True,
    )
    print("Policy attached successfully")

    # Install the EBS CSI driver addon
    print("Creating EBS CSI driver addon...")
    subprocess.run(
        ["aws", "eks", "create-addon",
         "--cluster-name", cluster_name,
         "--addon-name", "aws-ebs-csi-driver",
         "--region", region],
        check=True,
    )

    # Wait for addon to become ACTIVE
    print("Waiting for addon to become ACTIVE...")
    max_attempts = 30
    for attempt in range(max_attempts):
        result = subprocess.run(
            ["aws", "eks", "describe-addon",
             "--cluster-name", cluster_name,
             "--addon-name", "aws-ebs-csi-driver",
             "--region", region,
             "--query", "addon.status",
             "--output", "text"],
            capture_output=True,
            text=True,
            check=True,
        )
        status = result.stdout.strip()
        print(f"Addon status: {status} (attempt {attempt + 1}/{max_attempts})")

        if status == "ACTIVE":
            print("EBS CSI driver addon is ACTIVE")
            break
        elif status in ["CREATE_FAILED", "DEGRADED"]:
            raise RuntimeError(f"Addon installation failed with status: {status}")

        time.sleep(10)
    else:
        raise RuntimeError("Addon installation timed out")

    # Set storage class for Scanner V4 DB
    os.environ["SCANNER_V4_DB_STORAGE_CLASS"] = "gp2"
    print("EBS CSI driver installation complete")


# set required test parameters
os.environ["ORCHESTRATOR_FLAVOR"] = "k8s"
os.environ["KUBERNETES_PROVIDER"] = "eks"

os.environ["ROX_RISK_REPROCESSING_INTERVAL"] = "15s"
os.environ["ROX_SENSOR_CONNECTION_RETRY_MAX_INTERVAL"] = "30s"

# Install EBS CSI driver before running tests
install_ebs_csi_driver()

make_qa_e2e_test_runner(cluster=AutomationFlavorsCluster()).run()

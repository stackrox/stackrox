from datetime import datetime
import os
import re
import subprocess


def popen_graceful_kill(cmd):
    log_print(f"Sending SIGTERM to {cmd.args}")
    cmd.terminate()
    try:
        cmd.wait(5)
        log_print("Terminated")
    except subprocess.TimeoutExpired as err:
        log_print(f"Exception raised waiting after SIGTERM to {cmd.args}, {err}")
        # SIGKILL if necessary
        log_print(f"Sending SIGKILL to {cmd.args}")
        cmd.kill()
        cmd.wait(5)
        log_print("Terminated")


def set_ci_shared_export(name, value):
    with open(os.path.join(os.environ["SHARED_DIR"], "shared_env"), "a", encoding="utf-8") as shared_env:
        shared_env.write(f"export {name}={value}\n")


def log_print(*args):
    now = datetime.now()
    time = now.strftime("%H:%M:%S")
    print(f"{time}:", *args)


def enable_sfa_for_ocp():
    """Enable the Fact (SFA) agent on OCP >= 4.16.

    SFA is not supported on older OCP versions. This function checks
    CLUSTER_FLAVOR_VARIANT to determine the OCP version and sets
    SFA_AGENT=true if the version is 4.16 or later.
    """
    try:
        ocp_variant = os.environ.get("CLUSTER_FLAVOR_VARIANT", "")
        expr = r"openshift-4-ocp/\w+-(?P<major>\d+)\.(?P<minor>\d+)"
        m = re.match(expr, ocp_variant)
        if m:
            major = int(m.group("major"))
            minor = int(m.group("minor"))
            if (major, minor) >= (4, 16):
                os.environ["SFA_AGENT"] = "true"
                log_print("Enabled SFA agent for OCP", ocp_variant)
    except (ValueError, TypeError, AttributeError) as ex:
        log_print(f"Could not identify OCP version, SFA is disabled: {ex}")

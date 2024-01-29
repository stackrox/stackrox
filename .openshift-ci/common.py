import os
import subprocess


def popen_graceful_kill(cmd):
    print(f"Sending SIGTERM to {cmd.args}")
    cmd.terminate()
    try:
        cmd.wait(5)
        print("Terminated")
    except subprocess.TimeoutExpired as err:
        print(f"Exception raised waiting after SIGTERM to {cmd.args}, {err}")
        # SIGKILL if necessary
        print(f"Sending SIGKILL to {cmd.args}")
        cmd.kill()
        cmd.wait(5)
        print("Terminated")


def set_ci_shared_export(name, value):
    with open(os.path.join(os.environ["SHARED_DIR"], "shared_env"), "a", encoding="utf-8") as shared_env:
        shared_env.write(f"export {name}={value}\n")

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

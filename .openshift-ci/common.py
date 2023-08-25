import subprocess


def popen_graceful_kill(cmd):
    print(f"Sending SIGTERM to {cmd.args}")
    cmd.terminate()
    try:
        cmd.wait(5)
        print(f"Terminated")
    except subprocess.TimeoutExpired as err:
        # SIGKILL if necessary
        print(f"Sending SIGKILL to {cmd.args}")
        cmd.kill()
        cmd.wait(5)
        print(f"Terminated")

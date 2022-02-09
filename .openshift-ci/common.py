import subprocess


def popen_graceful_kill(cmd):
    cmd.terminate()
    try:
        cmd.wait(5)
    except subprocess.TimeoutExpired as err:
        cmd.kill()
        cmd.wait(5)
        raise err

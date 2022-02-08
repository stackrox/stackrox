import subprocess


def popen_cleanup(cmd):
    cmd.terminate()
    try:
        cmd.wait(5)
    except subprocess.TimeoutExpired as err:
        cmd.kill()
        cmd.wait(5)
        raise err

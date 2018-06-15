package OrchestratorManager

class OrchestratorCommon {
    static COMMAND_EXEC_TIMEOUT = 30000

    static CommandResults runCommand(cmd, List<String> envVariables = null, Boolean print = false) {
        def stdOut = new StringBuffer()
        def stdErr = new StringBuffer()

        if(print) println ">>> RUNNING COMMAND: ${cmd.join(" ")}"

        def process = cmd.execute(envVariables, null)
        process.consumeProcessOutput(stdOut, stdErr)
        process.waitForOrKill(COMMAND_EXEC_TIMEOUT)

        if(process.exitValue() == 143) println "runCommand killed due to timeout (${COMMAND_EXEC_TIMEOUT} ms)"

        if(print) {
            if(stdOut.toString().trim() != "") println "Standard Output: ${stdOut.toString().trim()}"
            if(stdErr.toString().trim() != "") println "Standard Error: ${stdErr.toString().trim()}"
            println "Exit Value: ${process.exitValue()}"
        }

        return new CommandResults(exitValue: process.exitValue(),
                standardOutput: stdOut.toString(),
                standardError: stdErr.toString())
    }

    static List<String> convertCmdArgsToList(String[] commands) {
        List<String> cmdsList = new ArrayList<>()

        commands.each {
            def args = it
            def temp = ""
            args.split("\\s").each {
                def arg = it

                if(temp != "") {
                    temp += " " + arg
                    if(temp.endsWith(temp.charAt(0).toString())) {
                        cmdsList.add(temp)
                        temp = ""
                    }
                }
                else {
                    if ((arg.startsWith("'") || arg.startsWith("\"")) && !arg.endsWith(arg.charAt(0).toString())) {
                        temp = arg
                    }
                    else cmdsList.add(arg)
                }
            }
        }
        return cmdsList
    }
}

class CommandResults {
    def exitValue
    def standardOutput
    def standardError
}



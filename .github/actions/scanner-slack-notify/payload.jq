# Renders the Slack Block Kit payload for a failed workflow run.
# Args (see build-payload.sh): $mention $workflow $description $run_url
# $event $branch $inputs $jobs $total_failed

def slack_escape:
  gsub("&"; "&amp;") | gsub("<"; "&lt;") | gsub(">"; "&gt;");

def section(t): {type: "section", text: {type: "mrkdwn", text: t}};
def context(t): {type: "context", elements: [{type: "mrkdwn", text: t}]};

def trigger_line:
  "Trigger: \($event) • Branch: \($branch)"
  + (if ($inputs | type) == "object" and ($inputs | length) > 0 then
       " • Inputs: " + ($inputs | to_entries
         | map("\(.key)=\(.value | tostring | slack_escape)")
         | join(", "))
     else "" end);

def job_block:
  section(
    ":x: *\(.name | slack_escape)*"
    + (if .failed_step then " — step \"\(.failed_step | slack_escape)\"" else "" end)
    + " (<\(.html_url)|logs>)"
    + (if (.annotations | length) > 0 then
         "\n" + (.annotations | map("> " + slack_escape) | join("\n"))
       else "" end)
  );

{
  text: "\($mention) \($description): \($run_url)",
  blocks: (
    [
      section(":red_circle: \($mention) *<\($run_url)|\($workflow | slack_escape)>* failed: \($description | slack_escape)"),
      context(trigger_line)
    ]
    + ($jobs | map(job_block))
    + (if $total_failed > ($jobs | length) then
         [context("…and \($total_failed - ($jobs | length)) more failed job(s); see the run.")]
       else [] end)
    + (if $total_failed == 0 then
         [section("No failed jobs resolved via the API; see the <\($run_url)|run> for details.")]
       else [] end)
  )
}

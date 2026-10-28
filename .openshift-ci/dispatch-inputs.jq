def sensitive:
    tostring | test("token|password|passwd|secret|credential|private.?key|authorization|cookie|(^|[_-])auth([_-]|$)|access.?key"; "i");

def redact:
    if type == "object" then
        if ((.name? // "") | sensitive) and has("value") then .value = "[REDACTED]" else . end
        | with_entries(if (.key | sensitive) then .value = "[REDACTED]" else . end)
    else . end;

env as $environment
| [$environment | to_entries[] | select(.key | sensitive) | .value | select(length > 0)] as $secrets
| {
    cwd: $cwd,
    argv: $ARGS.positional,
    environment: ($environment | with_entries(
        .value |= (. as $value | try fromjson catch $value)
    ))
}
| walk(redact)
| walk(if type == "string" then
    reduce $secrets[] as $secret (. ; split($secret) | join("[REDACTED]"))
  else . end)

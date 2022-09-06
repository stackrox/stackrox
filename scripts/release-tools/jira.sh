#!/usr/bin/env bash

MAX_NUM_OF_VERIFIED_TICKETS_TO_SHOW=10

MENU_OPTIONS=(
  "Show tickets with FixVersions that are not done yet"
  "Show tickets that need to be verified on demo cluster"
  "Quit"
)

FIX_VERSION="3.72.0" # copy this from JIRA

main() {
  local action="${1}"
  PS3='Choose action: '
  if [[ -n "$action" ]]; then
    exec_option "$action"
    exit 0
  fi
  RED='\033[0;31m'
  NC='\033[0m' # No Color
  echo -e "${RED}WARNING:${NC} some of these scripts may be outdated, bleeding-edge, or not working. Read the code before you run them to be on the safe side."
  select ans in "${MENU_OPTIONS[@]}"
  do
    exec_option "$ans"
  done
}

exec_option() {
  local num_options="${#MENU_OPTIONS[@]}"
  local last_option="$((num_options-1))"
  case "$1" in
    "${MENU_OPTIONS[$last_option]}"|"$((last_option+1))"|q|Q) exit 0;;
    "${MENU_OPTIONS[0]}"|1) not_done_yet;;
    "${MENU_OPTIONS[1]}"|2) tickets_to_verify;;
    *) echo "invalid option: '$1'";;
  esac
}

# is_ticket_checked takes ticket name and returns true or false
# the list of verified tickets is curated manually - the release officer should add new entries based on conversations with people
is_ticket_checked() {
  local ticket="$1"
  [[ -n "$ticket" ]] || die "input required"
  declare -A arr=(
    # EXAMPLES - remove before starting new release
    ["ROX-0001"]='verified'
    ["ROX-0002"]='N/A'
  )
  if test -n "${arr[$ticket]}"; then
    echo "${arr[$ticket]}"
  else
    echo "$ticket not found"
  fi
}

tickets_to_verify() {
  local QRY
  read -r -d '' QRY <<EOF
  (project = ROX OR project = "Rox Services" OR project = "Red Hat Advanced Cluster Security" )
    AND fixVersion = "$FIX_VERSION"
    AND issuetype != "STORY"
    AND status = Done
    ORDER BY key ASC, status DESC, priority DESC
EOF

  echo "This is automatically-generated list of tickets that have set \`FixVersions=$FIX_VERSION\`."
  echo "I kindly ask the respective assignees to use the demo clusters and verify that their fixes and features are working correctly."
  echo "When you verify tickets, please ping me to update the list and place it in the bottom (non-ping) section."
  echo ""

  jira_output="$(call_jira "$QRY")"
  print_ticket_verification_summary "$jira_output"

  echo ""
  echo "(list generated with JQL: \`$(echo "$QRY" | tr -d '\n')\`)"
}

not_done_yet() {
  local QRY
  read -r -d '' QRY <<EOF
  (project = ROX OR project = "Rox Services" OR project = "Red Hat Advanced Cluster Security" )
    AND component != "Documentation"
    AND component != "ACS Managed Service"
    AND fixVersion = "$FIX_VERSION"
    AND status != Done
    AND status != "Release Pending"
    AND status != CLOSED
    ORDER BY created DESC
EOF

  echo "This is automatically-generated list of tickets that have set \`FixVersions=$FIX_VERSION\` but are not done yet."
  echo "Respective assignees are kindly asked to finish the tickets soon."
  echo "Tickets marked as done will not appear on this list - the list has been generated automatically."
  echo ""

  jira_output="$(call_jira "$QRY")"
  print_ticket_todo_summary "$jira_output"

  echo ""
  echo "(list generated with JQL: \`$(echo "$QRY" | tr -d '\n')\`)"
}

call_jira() {
  local JQL="${1}"
  [[ -n "$JIRA_TOKEN" ]] || die "JIRA_TOKEN undefined"
  [[ -n "$JQL" ]] || die "JQL undefined"
  curl \
    -sSL \
    --get \
    --data-urlencode "jql=${JQL}" \
    -H "Authorization: Bearer $JIRA_TOKEN" \
    -H "Content-Type: application/json" \
    "https://issues.redhat.com/rest/api/2/search" | jq -r '.issues[] | "assingee: \"" + (.fields.assignee.displayName // "unassigned")  + "\" key: \"" + .key + "\""' | sort
}

print_ticket_todo_summary() {
  local todo=()
  while IFS= read -r line
  do
    ## take some action on $line
    regex='^assingee: "(.*)" key: "(.*)"$'
    if [[ $line =~ $regex ]]; then
      slack="$(name2slack "${BASH_REMATCH[1]}")"
      ticket="https://issues.redhat.com/browse/${BASH_REMATCH[2]}"
      todo+=("- @${slack}: $ticket")
    else
      echo "$line"
    fi
  done < <(echo "$1")
  printf "\n%s" "${todo[@]}"
  printf "\n"
}

print_ticket_verification_summary() {
  local verified=()
  local still_to_verify=()
  while IFS= read -r line
  do
    ## take some action on $line
    regex='^assingee: "(.*)" key: "(.*)"$'
    if [[ $line =~ $regex ]]; then
      slack="$(name2slack "${BASH_REMATCH[1]}")"
      ticket="https://issues.redhat.com/browse/${BASH_REMATCH[2]}"

      check="$(is_ticket_checked "${BASH_REMATCH[2]}")"
      case "$check" in
        "verified") verified+=("- ${slack}: $ticket :white_check_mark:");;
        "N/A") verified+=("- ${slack}: $ticket \`N/A\`");;
        *) still_to_verify+=("- @${slack}: $ticket");;
      esac
    else
      echo "$line"
    fi
  done < <(echo "$1")

  printf "Tickets that still need verification on the demo cluster: \n"
  printf "\n%s" "${still_to_verify[@]}"
  local num_verified="${#verified[@]}"
  printf "\n\nAlready verified tickets :gratitude-thank-you: or tickets that don't require verification on the demo cluster:\n"
  if (( num_verified < MAX_NUM_OF_VERIFIED_TICKETS_TO_SHOW )); then
    printf "\n%s" "${verified[@]}"
  else
    printf "\n%s" ":sunglasses: $num_verified items have been hidden to not obstruct the view"
  fi
  printf "\n"
}

name2slack() {
  # data taken from: https://stack-rox.atlassian.net/wiki/spaces/StackRox/pages/1620214005/Team+Alignments
  while [[ "$#" -gt 0 ]]; do
    case $1 in
        # sorted by firstname
        'Alexander Rukletsov') echo 'alexr';;
        'Daniel Haus') echo 'daniel';;
        'Cong Du') echo 'cong';;
        'Connor Gorman') echo 'gorman';;
        'Dmitrii Dolgov') echo 'dmitrii';;
        'Evan Benshetler') echo 'evan';;
        'Frederico Ramos Bittencourt') echo 'fred';;
        'Giles Hutton') echo 'giles';;
        'Ivan Degtiarenko') echo 'ivan';;
        'Jouko Virtanen') echo 'jouko';;
        'Juan Rodríguez Hortalá') echo 'jrodrig';;
        'Juan Rodriguez Hortala') echo 'jrodrig';;
        'Khushboo Sancheti') echo 'boo';;
        'Linda Song') echo 'linda';;
        'Luis Valero Martin') echo 'luis';;
        'Mandar Darwatkar') echo 'mandar';;
        'Marcin Owsiany') echo 'porridge';;
        'Mark Pedrotti') echo 'mark';;
        'Matthias Meidinger') echo 'matthias';;
        'Mauro Ezequiel Moltrasio') echo 'mauro';;
        'Michaël Petrov') echo 'Michaël';;
        'Misha Sugakov') echo 'misha';;
        'Moritz Clasmeier') echo 'moritz';;
        'Nakul C') echo 'nakul';;
        'Oscar Ward') echo 'oscar_ward';;
        'Piotr Rygielski') echo 'piotr';;
        'Robby Cochran') echo 'robby';;
        'Ross Tannenbaum') echo 'ross';;
        'Saif Chaudhry') echo 'saif';;
        'Simon Baumer') echo 'simon';;
        'Simon Bäumer') echo 'simon';;
        'Stephan Hesselmann') echo 'stephan';;
        'Tomasz Janiszewski') echo 'Tomek';;
        'Van Wilson') echo 'Van Wilson';;
        'Yann Brillouet') echo 'yann';;
        *) echo "'$1'"
    esac
    shift
  done
}

die() {
  >&2 echo "$@"
  exit 1
}

require_binary() {
  command -v "${1}" > /dev/null || die "Install ${1}"
}

main "$@"

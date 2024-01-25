#!/usr/bin/env bash

# This script returns predefined StackRox Slack Member ID for engineers of the StackRox team provided their GitHub
# login. Whenever a new member joins the team, please update the mapping in this script.

set -euo pipefail

if (( $# != 1 )) ; then
  echo "Usage: $0 <github-login>"
  exit 2
fi

github_login="$1"
slack_user=''

# You can find Slack Member ID by clicking on the user profile in Slack, then three dots, then Copy member ID.
# More info: https://api.slack.com/reference/surfaces/formatting#mentioning-users
# Here you can find GitHub logins: https://github.com/orgs/stackrox/people
# TODO: please keep the list in the alphabetic order. It is simpler to maintain it this way.
case "$github_login" in
'0x656b694d')       slack_user='U02MJ72K1B5' ;;
'BradLugo')         slack_user='U042Z3TSZU3' ;;
'GrimmiMeloni')     slack_user='U048VH2JZ1C' ;;
'JoukoVirtanen')    slack_user='U033Y28GYN4' ;;
'Maddosaurus')      slack_user='U01Q5L5R0GJ' ;;
'Molter73')         slack_user='U02A292NPV2' ;;
'RTann')            slack_user='U01NZ6U730X' ;;
'SimonBaeumer')     slack_user='U01Q5RMEHCK' ;;
'Stringy')          slack_user='U02KJKREKPY' ;;
'alanonthegit')     slack_user='U01PZFFSZRB' ;;
'alwayshooin')      slack_user='U01PLAWUU8N' ;;
'bradr5')           slack_user='U03UQ9DM44U' ;;
'c-du')             slack_user='U02NE59PHT3' ;;
'charmik-redhat')   slack_user='U035YKHMXEW' ;;
'clickboo')         slack_user='U01PFFU0YKD' ;;
'connorgorman')     slack_user='U01P76F3BFG' ;;
'dashrews78')       slack_user='U03FB5XE10V' ;;
'daynewlee')        slack_user='U03J855QWHF' ;;
'dcaravel')         slack_user='U04DF45CXBJ' ;;
'dhaus67')          slack_user='U02CZGZRGBF' ;;
'dvail')            slack_user='U032WL9RM53' ;;
'ebensh')           slack_user='U01Q7HTJ126' ;;
'erthalion')        slack_user='U02SV8VE3K3' ;;
'fredrb')           slack_user='U02H0SPRHC0' ;;
'gaurav-nelson')    slack_user='U01P6PMFGKF' ;;
'gavin-stackrox')   slack_user='U01QJF2LM9D' ;;
'house-d')          slack_user='U03H69TFKH9' ;;
'ivan-degtiarenko') slack_user='U01U2UKP7D4' ;;
'janisz')           slack_user='U0218FUVDMJ' ;;
'johannes94')       slack_user='U03E2SD2ZPB' ;;
'jschnath')         slack_user='U03AA9E6B09' ;;
'jvdm')             slack_user='U02TTV416HY' ;;
'keyallis')         slack_user='U02UX82JZAM' ;;
'kovayur')          slack_user='U033ZSBGEUQ' ;;
'ksurabhi91')       slack_user='U043ZP4RN76' ;;
'kurlov')           slack_user='U035001CQCV' ;;
'kylape')           slack_user='UGJML86DD'   ;;
'ludydoo')          slack_user='U04TFDR57KQ' ;;
'lvalerom')         slack_user='U02SJTV567N' ;;
'md2119')           slack_user='U01Q57U4JFK' ;;
'mfosterrox')       slack_user='U01PMH71ACU' ;;
'misberner')        slack_user='U01PFA6NUG3' ;;
'msugakov')         slack_user='U020QJZCQAH' ;;
'mtesseract')       slack_user='U02DKH1LQ5N' ;;
'mtodor')           slack_user='U039LQ48PT7' ;;
'ovalenti')         slack_user='U03F2F9EXUL' ;;
'pedrottimark')     slack_user='U01RN8V8DEH' ;;
'porridge')         slack_user='U020XCUG2LA' ;;
'rhybrillou')       slack_user='U02GPRG4NHF' ;;
'robbycochran')     slack_user='U03NAEPKDE1' ;;
'rukletsov')        slack_user='U01G6P17RTK' ;;
'sachaudh')         slack_user='U01QLCGS0NM' ;;
'sbostick')         slack_user='U01Q6UL38FM' ;;
'srcporter')        slack_user='U01P0CZ5QDV' ;;
'stehessel')        slack_user='U02SDMERUFP' ;;
'sthadka')          slack_user='U029PASTL5C' ;;
'theencee')         slack_user='U01Q57JQ3CZ' ;;
'tommartensen')     slack_user='U040F2EG19U' ;;
'vikin91')          slack_user='U02L405V2GH' ;;
'vjwilson')         slack_user='U01PKQQF0KY' ;;
'vladbologa')       slack_user='U03NFNXKPH9' ;;
'vulerh')           slack_user='U02A9CAR59T' ;;
esac

echo "${slack_user}"

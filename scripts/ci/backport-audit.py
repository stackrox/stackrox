#!/usr/bin/env python3
"""
Audit backport PRs and Jira issues for consistency and completeness.
"""

import argparse
import base64
import json
import os
import re
import subprocess
import sys
import traceback
from dataclasses import dataclass
from datetime import datetime
from typing import Dict, Optional

VERSION = "1.0.0"

# Slack user ID mapping (GitHub login -> Slack member ID)
# Source: scripts/ci/get-slack-user-id.sh
SLACK_USER_MAP: Dict[str, str] = {
    '0x656b694d': 'U02MJ72K1B5',
    'BradLugo': 'U042Z3TSZU3',
    'GrimmiMeloni': 'U048VH2JZ1C',
    'JoukoVirtanen': 'U033Y28GYN4',
    'Maddosaurus': 'U01Q5L5R0GJ',
    'Molter73': 'U02A292NPV2',
    'RTann': 'U01NZ6U730X',
    'SimonBaeumer': 'U01Q5RMEHCK',
    'Stringy': 'U02KJKREKPY',
    'ajheflin': 'U087GT2H45Q',
    'akameric': 'U076CG62KL4',
    'AlexVulaj': 'U03M3QKBES2',
    'alanonthegit': 'U01PZFFSZRB',
    'alwayshooin': 'U01PLAWUU8N',
    'bradr5': 'U03UQ9DM44U',
    'c-du': 'U02NE59PHT3',
    'charmik-redhat': 'U035YKHMXEW',
    'clickboo': 'U01PFFU0YKD',
    'dashrews78': 'U03FB5XE10V',
    'daynewlee': 'U03J855QWHF',
    'dcaravel': 'U04DF45CXBJ',
    'dvail': 'U032WL9RM53',
    'ebensh': 'U01Q7HTJ126',
    'erthalion': 'U02SV8VE3K3',
    'gaurav-nelson': 'U01P6PMFGKF',
    'guzalv': 'U08NQKQJH4N',
    'house-d': 'U03H69TFKH9',
    'janisz': 'U0218FUVDMJ',
    'johannes94': 'U03E2SD2ZPB',
    'jschnath': 'U03AA9E6B09',
    'jvdm': 'U02TTV416HY',
    'kovayur': 'U033ZSBGEUQ',
    'ksurabhi91': 'U043ZP4RN76',
    'kurlov': 'U035001CQCV',
    'kylape': 'UGJML86DD',
    'ludydoo': 'U04TFDR57KQ',
    'lvalerom': 'U02SJTV567N',
    'mclasmeier': 'U02DKH1LQ5N',
    'mfosterrox': 'U01PMH71ACU',
    'msugakov': 'U020QJZCQAH',
    'mtodor': 'U039LQ48PT7',
    'ovalenti': 'U03F2F9EXUL',
    'parametalol': 'U02MJ72K1B5',
    'pedrottimark': 'U01RN8V8DEH',
    'porridge': 'U020XCUG2LA',
    'rhybrillou': 'U02GPRG4NHF',
    'robbycochran': 'U03NAEPKDE1',
    'rukletsov': 'U01G6P17RTK',
    'sachaudh': 'U01QLCGS0NM',
    'stehessel': 'U02SDMERUFP',
    'sthadka': 'U029PASTL5C',
    'tommartensen': 'U040F2EG19U',
    'vikin91': 'U02L405V2GH',
    'vjwilson': 'U01PKQQF0KY',
    'vladbologa': 'U03NFNXKPH9',
    'vulerh': 'U02A9CAR59T',
}


def get_slack_mention(github_login: str) -> str:
    """
    Get Slack mention for GitHub user.

    Args:
        github_login: GitHub username

    Returns:
        Slack mention string (<@ID>, @username, or :konflux:)
    """
    if github_login == 'app/red-hat-konflux':
        return ':konflux:'

    slack_id = SLACK_USER_MAP.get(github_login)
    if slack_id:
        return f'<@{slack_id}>'
    return f'@{github_login}'


def main():
    print(f"Backport Audit Tool v{VERSION}")
    # Test Slack mentions
    print(f"janisz: {get_slack_mention('janisz')}")
    print(f"unknown: {get_slack_mention('unknown-user')}")
    print(f"konflux: {get_slack_mention('app/red-hat-konflux')}")
    sys.exit(0)


if __name__ == "__main__":
    main()

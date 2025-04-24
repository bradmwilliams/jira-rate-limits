#!/usr/bin/env python

import datetime
import json
import os
import sys
import time

import requests


def jira_request(method: str, url: str, **kwargs):
    print(f'[{datetime.datetime.now()}] Executing {method} request to {url}')
    response = requests.request(method, url, **kwargs)

    print(f'Status Code: {response.status_code}')
    limit = response.headers.get('x-ratelimit-limit')
    remaining = response.headers.get('x-ratelimit-remaining')
    interval = response.headers.get('x-ratelimit-interval-seconds')
    fillrate = response.headers.get('x-ratelimit-fillrate')
    retry_after = response.headers.get('retry-after')

    print('Jira Rate Limit Summary:')
    print(f'\t     Tokens: {remaining}/{limit} available')
    print(f'\t  Fill Rate: {fillrate} tokens per {interval} seconds')
    print(f'\tRetry After: {retry_after}')

    return response, retry_after


# Implementing the "Specific timed backoff" technique suggested here:
# https://confluence.atlassian.com/adminjiraserver0915/adjusting-your-code-for-rate-limiting-1402411093.html
def get_issue_with_retries(endpoint: str, issue_id: str, max_retries=5, **kwargs):
    retry_count = 0
    while True:
        response, retry_after = jira_request("GET", f'{endpoint}/rest/api/2/issue/{issue_id}', **kwargs)
        match response.status_code:
            case 200:
                return json.loads(response.text)
            case 429:
                retry_count += 1
                if retry_count >= max_retries:
                    print(f'Maximum retries ({max_retries}) exceeded')
                    return {}
                print(f'Sleeping for: {retry_after} seconds\n')
                time.sleep(int(retry_after))
            case _:
                print(f'An unexpected error {response.status_code} has occurred')
                sys.exit(1)


if __name__ == '__main__':
    jira = os.getenv('JIRA_ENDPOINT')
    if jira is None:
        print('Environment variable "JIRA_ENDPOINT" not set!')
        sys.exit(1)

    api_key = os.getenv('JIRA_API_KEY')
    if api_key is None:
        print('Environment variable "JIRA_API_KEY" not set!')
        sys.exit(1)

    headers = {
        'Authorization': f'Bearer {api_key}',
    }
    issue = get_issue_with_retries(jira, 'OCPBUGS-36344', headers=headers)
    if 'id' in issue:
        print(f'Successfully retrieved issue: {issue["id"]}')

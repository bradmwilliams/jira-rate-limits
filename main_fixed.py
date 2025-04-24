#!/usr/bin/env python

import datetime
import json
import math
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
    node_id = response.headers.get('x-anodeid')
    rate_limit = int(int(interval) / int(fillrate))

    print('Jira Rate Limit Summary:')
    print(f'       Node: {node_id}')
    print(f'     Tokens: {remaining}/{limit} available')
    print(f'  Fill Rate: {fillrate} tokens per {interval} seconds = {rate_limit}s')
    print(f'Retry After: {retry_after} seconds on node: {node_id}')

    return response, retry_after, node_id, rate_limit


# Implementing the "Specific timed backoff" technique suggested here:
# https://confluence.atlassian.com/adminjiraserver0915/adjusting-your-code-for-rate-limiting-1402411093.html
def get_issue_with_retries(endpoint: str, issue_id: str, max_retries=3, **kwargs):
    retry_count = 0
    wait_time_per_node = {}
    while True:
        response, retry_after, node_id, rate_limit = jira_request("GET", f'{endpoint}/rest/api/2/issue/{issue_id}', **kwargs)
        match response.status_code:
            case 200:
                return json.loads(response.text)
            case 429:
                wait_time_per_node[node_id] = int(retry_after)
                print(f'wait_time_per_node: {wait_time_per_node}')
                if 0 == sum(wait_time_per_node.values()):
                    time.sleep(rate_limit)  # wait to renew new token request
                    retry_count += 1
                elif retry_count >= max_retries:
                    print(f'Maximum retries ({max_retries}) exceeded')
                    return {}
                else:
                    sleep = simple_backoff(1, 30, retry_count)
                    print(f'Sleeping for: {sleep} seconds\n')
                    time.sleep(sleep)
            case _:
                print(f'An unexpected error {response.status_code} has occurred')
                sys.exit(1)


def simple_backoff(min_wait, max_wait, attempt_num):
    sleep = math.pow(2, attempt_num) * min_wait
    if sleep > max_wait:
        sleep = max_wait
    return sleep


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

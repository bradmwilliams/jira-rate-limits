# jira-rate-limits

### Details

I have been investigating an odd behavior, that we've observed when interacting with Jira, that has resulted in our API_TOKEN being rate-limited for "excessive API calls".

### Contents

* [main.py](./main.py): Script that makes a `GET` request to the `/rest/api/2/issue/{issue_id}` endpoint, displays the various rate-limit headers returned by Jira, and takes the "
  Specific timed backoff" approach (#1i)
  when it receives a status code `429` in the response.

### References

1. Jira's reference documentation on how to handle being rate limited
    1. [Adjusting your code for rate limiting](https://confluence.atlassian.com/adminjiraserver0915/adjusting-your-code-for-rate-limiting-1402411093.html)
2. The third-party library that [Prow's Jira Integration](https://github.com/kubernetes-sigs/prow/tree/main/pkg/jira) uses to handle calls into Jira
    1. [go-retriablehttp DefaultBackup logic](https://github.com/hashicorp/go-retryablehttp/blob/390c1d807b1dfda09c64e992bdd5e58a00daa698/client.go#L544-L601)
3. The `429 Too Many Requests` specification
    1. https://www.rfc-editor.org/rfc/rfc6585#section-4

### Solutions
The team responsible for our Jira installation was able to reproduce the issue and provide a customized solution that works with the existing limitations of our jira environment.

The recommended solution, with a few modifications by me, is located here:  [main_fixed.py](./main_fixed.py)

I have ported the solution over to Go, for usage in a few of our existing tools.  The main logic, for handling the rate limiting can be found here: [rate-limiting.go](./pkg/jira/rate-limiting.go)

I wrote a couple different test harnesses to exercise the logic:
* [jira-prow-tester](./cmd/jira-prow-tester/main.go):  I made the necessary changes, in [Prow](https://github.com/kubernetes-sigs/prow), that allows consumers to specify their own custom Backoff logic, instead of relying on the [DefaultBackoff](https://github.com/hashicorp/go-retryablehttp/blob/390c1d807b1dfda09c64e992bdd5e58a00daa698/client.go#L551-L566) provided as part of `hashicorp/go-retryablehttp`.  This command configures prow, specifies our own `CustomBackoff` function and issues a couple different calls to Jira.  It can be executed like:   
  `$ jira-prow-tester --jira-bearer-token-file </path/to/file> --jira-endpoint <JIRA ENDPOINT>`


* [jira-tester](./cmd/jira-tester/main.go): This file is just a scaled down, without the overhead of Prow, version of the code above.  It can executed like: 
  ```shell
  export JIRA_API_KEY=<TOKEN>
  export JIRA_ENDPOINT=<JIRA_ENDPOINT>
  jira-tester
  ```

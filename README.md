# Noty

### Introduction
Someone is waiting for the notification that will be sent automatically.

#### Functionality:
* Observation how people from jira logging time.
> You can observe how logging time some people from jira whose listed `.env` as `JIRA_EMAILS_FOR_OBSERVATION` with comma separated.
* Send notification to slack channel.
> If you want to ask them logging time then just click to `ask logging`. People which aren't over `JIRA_THRESHOLD_HOURS` won't get notice. 

----
System requirements:
* OSX 12.2.1 and higher

### Installation
Build from the source:
```shell
git clone https://github.com/revilon1991/noty.git
cd noty
cp .env.dist .env
# here you should set your values of variables to .env
make
```
_It's expected that [Make](https://www.gnu.org/software/make/) was installed in your operating system._

### Usage
1. Run `bin/Noty.app`.
2. Click to ![Noty](./Resources/icon-xss.png "Noty") from menu bar.

### Additional
* The `SLACK_TOKEN` generates by [this](https://slack.com/help/articles/215770388-Create-and-regenerate-API-tokens#custom-or-third-party-app-tokens) instructions.
    * You make an application and install that to your workspace by above instractions after that you'll see token.
* The `JIRA_API_TOKEN` generates [here](https://id.atlassian.com/manage-profile/security/api-tokens)

License
-------

[![license](https://img.shields.io/badge/License-MIT-green.svg?style=flat-square)](./LICENSE)

#!/bin/bash

export CIRCLE_TOKEN_MHASHER='73ca42abd09fb9dd7591e70f65f897c26404c547'
export CIRCLE_TOKEN_URLARBITRATOR='43de5d9653835b09959f309954525b4b9b28ec8a'
export CIRCLE_TOKEN_SCHEDULER='b959d2fe6ad9cc630d5ba2b4cb129a800f544e8c'

curl -H "Circle-Token: $CIRCLE_TOKEN_URLARBITRATOR" https://circleci.com/api/v1.1/project/github/HPISTechnologies/urlarbitrator-engine/latest/artifacts \
   | grep -o 'https://[^"]*' \
   | wget --verbose --header "Circle-Token: $CIRCLE_TOKEN_URLARBITRATOR" --input-file -

curl -H "Circle-Token: $CIRCLE_TOKEN_SCHEDULER" https://circleci.com/api/v1.1/project/github/HPISTechnologies/scheduling-engine/latest/artifacts \
   | grep -o 'https://[^"]*' \
   | wget --verbose --header "Circle-Token: $CIRCLE_TOKEN_SCHEDULER" --input-file -

curl -H "Circle-Token: $CIRCLE_TOKEN_MHASHER" https://circleci.com/api/v1.1/project/github/HPISTechnologies/mhasher-engine/latest/artifacts \
   | grep -o 'https://[^"]*' \
   | wget --verbose --header "Circle-Token: $CIRCLE_TOKEN_MHASHER" --input-file -
#!/bin/sh

OPT=""

if [ "$CHECK" == "true" ]; then
  OPT="$OPT --check"
fi

if [ "$INVENTORY_FILE" != "" ]; then
  OPT="$OPT -i $INVENTORY_FILE"
fi

if [ "$TARGET_USER" != "" ]; then
  OPT="$OPT -u $TARGET_USER"
fi

if [ "$PRIVATE_KEY" != "" ] || [ "$PRIVATE_KEY_FILE" != "" ]; then
  if [ "$PRIVATE_KEY" != "" ]; then
    echo "$PRIVATE_KEY" > /tmp/private_key
    chmod 0600 /tmp/private_key
    OPT="$OPT --private-key /tmp/private_key"
  elif [ "$PRIVATE_KEY_FILE" != "" ]; then
    OPT="$OPT --private-key $PRIVATE_KEY_FILE"
  fi
fi

if [ "$BECOME_PASS" != "" ] || [ "$SSH_PASS" != "" ]; then
  if [ "$BECOME_PASS" != "" ]; then
    echo "ansible_become_pass: $BECOME_PASS" >> /tmp/password.yml
  fi
  if [ "$SSH_PASS" != "" ]; then
    echo "ansible_ssh_pass: $SSH_PASS" >> /tmp/password.yml
  fi
  OPT="$OPT --extra-vars=@/tmp/password.yml"
fi

if [ "$VAULT_PASS" != "" ]; then
  echo "$VAULT_PASS" > /tmp/vault_pass.txt
  OPT="$OPT --vault-password-file /tmp/vault_pass.txt"
fi


set -ux
ansible-playbook $OPT $PLAYBOOK_FILE

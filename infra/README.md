# infra

Ansible playbook to bootstrap the Saftaja VPS.

## Setup

```bash
cp hosts.yml.example hosts.yml
# fill in your VPS IP

ansible-galaxy install -r requirements.yml
```

## Running

```bash
# first time (fresh VPS)
MY_PUBLIC_KEY="$(cat ~/.ssh/your_key.pub)" \
ansible-playbook -i hosts.yml bootstrap-vps.yml -u root

# after bootstrap
MY_PUBLIC_KEY="$(cat ~/.ssh/your_key.pub)" \
ansible-playbook -i hosts.yml bootstrap-vps.yml -u maintenance
```

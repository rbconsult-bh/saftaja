# infra

here we have ansible to bootstrap and update the vps and track it in git for ease of reproducability :D

## how to apply the changes?

first ensure you copy `hosts.yml.example` to `hosts.yml` and fill in the vps ip.

then run the requirements install command:
```bash
ansible-galaxy install -r requirements.yml
```

then finally run:

```bash
MY_PUBLIC_KEY="$(cat ~/.ssh/your_key.pub)" \
> ansible-playbook -i hosts.yml bootstrap-vps.yml -u root
```


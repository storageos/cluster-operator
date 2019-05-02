# Setup OpenShift Test Environment

`Vagrantfile` contains configuration for creating an ubuntu trusty VM with
enough memory and diskspace to setup openshift and run the operator tests.

Create the VM:
```console
$ vagrant up
```

Once the VM is ready, ssh into the VM and run the setup scripts in it to setup
everything. Read the comments in the scripts for reasons why things are done
in certain ways. Most of the parts of the scripts is a combination of the travis
CI configuration file and e2e.sh file, but only the parts that apply to
openshift setup.

```console
$ vagrant ssh
$ bash /vagrant/openshift-seutp-1.sh

# Re-login
$ exit
$ vagrant ssh

# Continue setup
$ bash /vagrant/openshift-seutp-2.sh
```

# Rancher AWS host cleanup

### Info
This repo contains the code to build a docker container that runs under a Rancher environment and removes terminiated AWS hosts.

The container is alpine based and utilises the go AWSSDK.

An AWS key and secret are required with permissions to describe instances in order for it to validate the host status.
An assumption is made that the hostname of the host to be checked is the private DNS name assigned by Amazon. The second element of the DNS name would then be the region that the host belongs to.
If this name is different then the host will not be removed from Rancher. Hosts from other providers will generate an error in the log but will be otherwise untouched.

### Usage
To use the build container please use the library entry in the Rancher Community library

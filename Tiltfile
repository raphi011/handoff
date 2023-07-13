docker_build('handoff', '.', dockerfile='./Dockerfile')
k8s_yaml('./deployment.yaml')
k8s_resource('handoff', port_forwards=1337)

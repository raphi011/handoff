docker_build('raphi011/handoff-example', '.', dockerfile='./Dockerfile')
k8s_yaml(helm('./chart', 'handoff'))
k8s_resource('handoff', port_forwards=1337)

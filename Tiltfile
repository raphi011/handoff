docker_build('raphi011/handoff-example', '.', dockerfile='./Dockerfile')
k8s_yaml(helm('./chart', 'handoff'))
k8s_resource('handoff', port_forwards=1337)

k8s_yaml(['./test_cluster/elasticsearch_deploy.yaml', './test_cluster/logstash_deploy.yaml', './test_cluster/logstash_cm.yaml'])

# Port-forwards to access services locally
k8s_resource('elasticsearch', port_forwards=9200)
k8s_resource('logstash', port_forwards=5044)

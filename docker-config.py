import sys
from jinja2 import Template


def generate_clients(number_of_clients):
    clients = {}
    for i in range(1,number_of_clients + 1):
        clients[f"client{i}"] = {
            'container_name': f'client{i}',
            'image': 'client:latest',
            'entrypoint': '/client',
            'environment': [
                f'CLI_ID={i}',
                'CLI_LOG_LEVEL=DEBUG'
            ],
            'networks': [
                'testing_net'
            ],
            'depends_on': [
                'server'
            ]
        }
    return clients

def write_yaml_file(number_of_clients):
    with open('compose-template.yaml', 'r') as template_docker_compose:
        template = template_docker_compose.read()

    template = Template(template)        
    rendered_yaml = template.render(clients=generate_clients(number_of_clients))

    with open('docker-compose-dev.yaml', 'w') as yaml_file:
        yaml_file.write(rendered_yaml)


if __name__ == "__main__":
    if len(sys.argv) == 2:
        arg = sys.argv[1]
        try:
            number_of_clients = int(arg)
        except ValueError:
            print("The number of clients should be an integer")
            sys.exit(1)            
                 
    else:
        print("Wrong number of arguments. Please provide only one argument")
        sys.exit(1)
        
    print(f"Selected number of clients: {number_of_clients}")
    write_yaml_file(number_of_clients)
    
    
        

from fabric.api import *
import requests

project = "broome"
repository = "git@github.com:Bowery/" + project + ".git"
hosts = [
  'ubuntu@ec2-54-82-96-245.compute-1.amazonaws.com'
]
env.key_filename = '/home/ubuntu/.ssh/id_aws'
env.password = 'java$cript'

def restart():
  with cd('/home/ubuntu/gocode/src/' + project):
    run('git pull')
    with cd('server'):
      sudo('GOPATH=/home/ubuntu/gocode go get -d')
      sudo('GOPATH=/home/ubuntu/gocode go build')
      run('myth static/style.css static/out.css')

    sudo('cp -f ' + project + '.conf /etc/init/' + project + '.conf')
    sudo('initctl reload-configuration')
    sudo('restart ' + project)

def deploy():
  execute(restart, hosts=hosts)

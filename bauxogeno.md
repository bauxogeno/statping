# run go build
from project root folder type:  
`go build -o statping ./cmd`  
Will create file with name statping, this is a binary go file.

# Create base image  
Use vscode shortcut command or type the command below:  
`docker build --pull --rm -f "Dockerfile.base" -t statping:base "."`

# Create final image  
execute this command from command line, not from node because it send pull=true to force the pull image from docker repo, that we dont.  
`docker build --pull --rm -f "Dockerfile" -t statping:latest "." --pull=false`

# Hot reload for frontend
start backend listening on port 8585  
`$ statping -p 8585`  
  
go to frontend folder and launch dev script  
`$ cd frontend`  
`$ npm run dev`  
#!/bin/bash

SESSION_NAME="handoff"
PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Check if the session already exists, attach if it does, otherwise create it
tmux has-session -t $SESSION_NAME 2>/dev/null
if [ $? != 0 ]; then
  # Create new tmux session with the working directory set
  tmux new-session -d -s $SESSION_NAME -c $PROJECT_DIR

  # Create a window for nvim
  tmux rename-window -t $SESSION_NAME:1 'nvim'
  tmux send-keys -t $SESSION_NAME:1 -c $PROJECT_DIR 'nvim' C-m

  # Create a window for running commands
  tmux new-window -t $SESSION_NAME:2 -n 'run' -c $PROJECT_DIR
  tmux send-keys -t $SESSION_NAME:2 'echo "Use this window to run commands"' C-m

  # Create a window for running the backend server
  tmux new-window -t $SESSION_NAME:3 -n 'backend' -c $PROJECT_DIR
  tmux send-keys -t $SESSION_NAME:3 'go run ./cmd/example-server-bootstrap/' C-m
fi

# Switch back to nvim window
tmux select-window -t $SESSION_NAME:1

# Press 's' key to continue with the nvim session
tmux send-keys -t $SESSION_NAME:1 's' C-m

# Attach to the session
tmux attach-session -t $SESSION_NAME

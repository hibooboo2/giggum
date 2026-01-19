#!/bin/bash

# Build script for giggum Docker image
set -e

# Configuration
IMAGE_NAME="giggum"
IMAGE_TAG="latest"
CONTAINER_NAME="giggum-test"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}Building giggum Docker image...${NC}"

# Check if Docker is available
if ! command -v docker &> /dev/null; then
    echo -e "${RED}Error: Docker is not installed or not in PATH${NC}"
    exit 1
fi

# Check if we're in the right directory
if [ ! -f "Dockerfile" ]; then
    echo -e "${RED}Error: Dockerfile not found in current directory${NC}"
    echo "Please run this script from the directory containing the Dockerfile"
    exit 1
fi

# Check if go.mod exists
if [ ! -f "go.mod" ]; then
    echo -e "${RED}Error: go.mod not found. This doesn't appear to be a Go project.${NC}"
    exit 1
fi

# Build the Docker image
echo -e "${YELLOW}Building Docker image: ${IMAGE_NAME}:${IMAGE_TAG}${NC}"
docker build -t "${IMAGE_NAME}:${IMAGE_TAG}" .

# Check if build was successful
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Docker image built successfully!${NC}"
else
    echo -e "${RED}✗ Docker image build failed!${NC}"
    exit 1
fi

# Display image information
echo -e "${BLUE}Docker image information:${NC}"
docker images | grep "${IMAGE_NAME}"

echo -e "${GREEN}Build completed successfully!${NC}"
echo -e "${YELLOW}To run the container:${NC}"
echo -e "  docker run -v \$(pwd):/src ${IMAGE_NAME}:${IMAGE_TAG}"
echo -e ""
echo -e "${YELLOW}To run with custom command:${NC}"
echo -e "  docker run -v \$(pwd):/src ${IMAGE_NAME}:${IMAGE_TAG} giggum -n 5"
echo -e ""
echo -e "${YELLOW}To test interactively:${NC}"
echo -e "  docker run -it -v \$(pwd):/src ${IMAGE_NAME}:${IMAGE_TAG} bash"

# Optional: Test the container
if [ "$1" = "--test" ]; then
    echo -e "${BLUE}Testing the container...${NC}"
    
    # Stop and remove any existing test container
    if docker ps -a --format 'table {{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
        echo -e "${YELLOW}Removing existing test container...${NC}"
        docker stop "${CONTAINER_NAME}" >/dev/null 2>&1 || true
        docker rm "${CONTAINER_NAME}" >/dev/null 2>&1 || true
    fi
    
    # Run test container
    echo -e "${YELLOW}Starting test container...${NC}"
    docker run --name "${CONTAINER_NAME}" -v "$(pwd):/src" "${IMAGE_NAME}:${IMAGE_TAG}" giggum --help
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ Container test passed!${NC}"
    else
        echo -e "${RED}✗ Container test failed!${NC}"
        exit 1
    fi
    
    # Clean up test container
    docker rm "${CONTAINER_NAME}" >/dev/null 2>&1
fi
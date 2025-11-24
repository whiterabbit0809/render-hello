# Use a small Node base image
FROM node:20-alpine

# Create app directory
WORKDIR /app

# Copy package files and install (if any dependencies)
COPY package*.json ./
RUN npm install --only=production

# Copy the rest of the app
COPY . .

# Tell Render (and Docker) which port the app listens on
EXPOSE 3000

# Start the app
CMD ["npm", "start"]

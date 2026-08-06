# multi-stage build for React/Vite frontend
FROM node:22-alpine AS build

WORKDIR /app

# install dependencies (package-lock.json ensures reproducible installs)
COPY package.json package-lock.json ./
RUN npm ci

# build the production bundle
COPY . .
RUN npm run build

# runtime image: nginx serves the static bundle
FROM nginx:stable-alpine

# copy the built assets into nginx's web root
COPY --from=build /app/dist /usr/share/nginx/html

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
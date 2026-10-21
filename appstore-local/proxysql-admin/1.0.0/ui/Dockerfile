FROM node:20-alpine
WORKDIR /app
COPY package.json ./
RUN npm install --omit=dev --no-audit --no-fund --registry=https://registry.npmmirror.com
COPY server.js ./
COPY public ./public
ENV PANEL_DATA_DIR=/app/data
EXPOSE 3000
CMD ["node", "server.js"]

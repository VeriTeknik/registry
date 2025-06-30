#!/bin/bash

# Script to generate SSH deployment key for GitHub Actions

echo "🔑 Generating SSH deployment key for GitHub Actions..."

# Generate key
ssh-keygen -t ed25519 -f deploy_key -N "" -C "github-actions-deploy"

echo ""
echo "✅ SSH key generated successfully!"
echo ""
echo "📋 Next steps:"
echo ""
echo "1. Add the PUBLIC key to your server's authorized_keys:"
echo "   On your server, run:"
echo "   echo '$(cat deploy_key.pub)' >> ~/.ssh/authorized_keys"
echo ""
echo "2. Add the PRIVATE key to GitHub Secrets:"
echo "   - Go to your repository Settings → Secrets and variables → Actions"
echo "   - Click 'New repository secret'"
echo "   - Name: DEPLOY_KEY"
echo "   - Value: Copy the content below:"
echo ""
echo "--- BEGIN DEPLOY_KEY ---"
cat deploy_key
echo "--- END DEPLOY_KEY ---"
echo ""
echo "3. Remove the local key files after adding to GitHub:"
echo "   rm deploy_key deploy_key.pub"
echo ""
echo "⚠️  IMPORTANT: Never commit these keys to your repository!"
// Declarative Jenkins pipeline that tags and creates a GitHub release when main is pushed
pipeline {
  agent any

  // Only run pipeline steps on the main branch (works with Multibranch Pipeline or GitHub webhook triggers)
  // No top-level triggers: Multibranch Pipeline / webhooks control when this runs.

  options {
    skipDefaultCheckout(true)
    timestamps()
  }

  stages {
    stage('Checkout') {
      when { branch 'main' }
      steps {
        checkout scm
      }
    }

    stage('Determine next version') {
      when { branch 'main' }
      steps {
        // Calculate next patch semver tag. If no tag exists, start from v0.0.1
        // Use Groovy to fetch tags and compute the next version so we can reliably set
        // an env var (avoids writing/sourcing a file and needing stash/unstash).
        script {
          // ensure tags are available
          sh 'git fetch --tags'

          def latest = sh(script: 'git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0"', returnStdout: true).trim()
          echo "Latest tag: ${latest}"

          def ver = latest.replaceFirst(/^v/, '')
          def parts = ver.tokenize('.')
          def major = (parts.size() > 0) ? parts[0].toInteger() : 0
          def minor = (parts.size() > 1) ? parts[1].toInteger() : 0
          def patch = (parts.size() > 2) ? parts[2].toInteger() : 0
          patch = patch + 1

          env.NEW_TAG = "v${major}.${minor}.${patch}"
          echo "NEW_TAG=${env.NEW_TAG}"
        }
      }
    }

    stage('Tag and push') {
      when { branch 'main' }
      steps {
        withCredentials([
          usernamePassword(credentialsId: 'aa53f87f-dcf2-40cb-b44b-ed68bb9f0271', usernameVariable: 'GIT_USERNAME', passwordVariable: 'GIT_PASSWORD')
        ]) {
          sh '''
          set -eux
          # ensure NEW_TAG is present (set in previous stage via env.NEW_TAG)
          echo "Using NEW_TAG=${NEW_TAG}"
          if [ -z "${NEW_TAG}" ]; then
            echo "ERROR: NEW_TAG is not set" >&2
            exit 1
          fi

          # ensure git user
          git config user.email "jenkins@local"
          git config user.name "jenkins"

          # compute repo owner/name
          remote_url=$(git remote get-url origin)
          echo "Remote URL: ${remote_url}"
          # normalize to owner/repo.git for both ssh and https forms
          repo_path=$(echo "${remote_url}" | sed -E 's,git@[^:]+:,,' | sed -E 's,https?://[^/]+/,,')
          repo_path=${repo_path%.git}
          OWNER=$(echo "${repo_path}" | cut -d/ -f1)
          REPO=$(echo "${repo_path}" | cut -d/ -f2)

          git tag -a "$NEW_TAG" -m "Automated release $NEW_TAG"

          # push tag using credentials
          remote_auth="https://${GIT_USERNAME}:${GIT_PASSWORD}@github.com/${OWNER}/${REPO}.git"
          git push "$remote_auth" "$NEW_TAG"
          '''
        }
      }
    }

    stage('Create GitHub release') {
      when { branch 'main' }
      steps {
        withCredentials([
          string(credentialsId: 'aaf467cd-851b-4db5-aa58-5feb4e7bc942', variable: 'GITHUB_TOKEN')
        ]) {
          sh '''
          set -eux
          echo "Using NEW_TAG=${NEW_TAG}"
          if [ -z "${NEW_TAG}" ]; then
            echo "ERROR: NEW_TAG is not set" >&2
            exit 1
          fi

          remote_url=$(git remote get-url origin)
          repo_path=$(echo "${remote_url}" | sed -E 's,git@[^:]+:,,' | sed -E 's,https?://[^/]+/,,')
          repo_path=${repo_path%.git}
          OWNER=$(echo "${repo_path}" | cut -d/ -f1)
          REPO=$(echo "${repo_path}" | cut -d/ -f2)

          api_url="https://api.github.com/repos/${OWNER}/${REPO}/releases"

          payload=$(cat <<EOF
{"tag_name":"$NEW_TAG","name":"$NEW_TAG","body":"Automated release for $NEW_TAG","draft":false,"prerelease":false}
EOF
)

          curl -s -X POST -H "Authorization: token ${GITHUB_TOKEN}" -H "Content-Type: application/json" -d "$payload" "$api_url"
          '''
        }
      }
    }
  }

  post {
    success {
      echo 'Release created successfully.'
    }
    failure {
      echo 'Pipeline failed — check logs.'
    }
  }
}

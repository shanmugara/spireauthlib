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
        sh '''
        set -eux
        git fetch --tags
        latest=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
        echo "Latest tag: $latest"
        ver=${latest#v}
        # Parse the version components in a POSIX-compatible way (avoid bash-only '<<<')
        major=$(echo "$ver" | cut -d. -f1)
        minor=$(echo "$ver" | cut -d. -f2)
        patch=$(echo "$ver" | cut -d. -f3)
        major=${major:-0}
        minor=${minor:-0}
        patch=${patch:-0}
        patch=$((patch + 1))
        NEW_TAG="v${major}.${minor}.${patch}"
        echo "NEW_TAG=$NEW_TAG" > next_tag.env
        '''
        stash includes: 'next_tag.env', name: 'next-tag'
      }
    }

    stage('Tag and push') {
      when { branch 'main' }
      steps {
        unstash 'next-tag'
        withCredentials([
          usernamePassword(credentialsId: 'git-credentials', usernameVariable: 'GIT_USERNAME', passwordVariable: 'GIT_PASSWORD')
        ]) {
          sh '''
          set -eux
          . next_tag.env

          # ensure git user
          git config user.email "jenkins@local"
          git config user.name "jenkins"

          # compute repo owner/name
          remote_url=$(git remote get-url origin)
          echo "Remote URL: $remote_url"
          if echo "$remote_url" | grep -q "@" ; then
            # git@github.com:owner/repo.git -> take substring after ':'
            repo_path=${remote_url#*:}
          else
            # https://github.com/owner/repo.git -> remove protocol+host prefix
            repo_path=${remote_url#*://*/}
          fi
          # strip trailing .git if present
          repo_path=${repo_path%.git}
          OWNER=$(echo "$repo_path" | cut -d/ -f1)
          REPO=$(echo "$repo_path" | cut -d/ -f2)

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
          string(credentialsId: 'github-token', variable: 'GITHUB_TOKEN')
        ]) {
          sh '''
          set -eux
          . next_tag.env

          remote_url=$(git remote get-url origin)
          if echo "$remote_url" | grep -q "@" ; then
            repo_path=${remote_url#*:}
          else
            repo_path=${remote_url#*://*/}
          fi
          repo_path=${repo_path%.git}
          OWNER=$(echo "$repo_path" | cut -d/ -f1)
          REPO=$(echo "$repo_path" | cut -d/ -f2)

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

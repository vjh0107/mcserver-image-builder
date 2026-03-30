pipeline {
    agent any

    tools {
        go 'go-1.26'
    }

    stages {
        stage('Test') {
            steps {
                sh 'go build ./...'
                sh 'go test ./...'
            }
        }

        stage('Release') {
            when {
                buildingTag()
            }
            environment {
                GITHUB_TOKEN = credentials('github-token')
            }
            steps {
                sh 'goreleaser release --clean'
            }
        }
    }
}

package hook

// 触发事件
const (
	// EventsTypeComment 评论事件
	EventsTypeComment = "comment"
	// EventsTypePR pull request事件
	EventsTypePR = "pr"
	// EventsTypePush push事件
	EventsTypePush = "push"
	// EventsTypeBuild 手动运行事件
	EventsTypeBuild = "build"
	// EventsTypeRebuild 手动重新构建
	EventsTypeRebuild = "rebuild"
)

const (
	// Gitee webhook headers
	GiteeEvent          = "X-Gitee-Event"
	GiteeEventPush      = "Push Hook"
	GiteeEventNote      = "Note Hook"
	GiteeEventPR        = "Merge Request Hook"
	GiteeEventPROpen    = "open"
	GiteeEventPRUpdate  = "update"
	GiteeEventPRComment = "comment"
)

const (
	// GitHub webhook headers
	GithubEvent             = "X-GitHub-Event"
	GithubEventIssueComment = "issue_comment"
	GithubEventPush         = "push"
	GithubEventPR           = "pull_request"
	GithubEventPROpen       = "open"
	GithubEventPRUpdate     = "update"
	GithubEventPRComment    = "comment"
)

const (
	// GitLab webhook headers
	GitlabEvent     = "X-Gitlab-Event"
	GitlabEventPush = "Push Hook"
	GitlabEventPR   = "Merge Request Hook"
	GitlabEventNote = "Note Hook"
)

const (
	// Gitea webhook headers
	GiteaEvent     = "X-Gitea-Event"
	GiteaEventPush = "push"
	GiteaEventPR   = "pull_request"
	GiteaEventNote = "issue_comment"
)

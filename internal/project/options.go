package project

type Options struct {
	DryRun, Yes, IgnoreScripts, NoInstall, NoServices, InstallServices bool
	CloneCompanions, NoCompanions                                      bool
	Verify, VerifyCmd, NoBuild                                         bool
	Timeout                                                            int
	Setup, StatusCmd, Resume                                           bool
	Url, Dir                                                           string
}

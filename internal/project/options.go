package project

type Options struct {
	DryRun, Yes, IgnoreScripts, NoInstall, NoServices, InstallServices bool
	CloneCompanions, NoCompanions                                      bool
	Verify, VerifyCmd, NoBuild                                         bool
	Timeout                                                            int
	Setup, StatusCmd, Resume                                           bool
	CI, JSON                                                           bool
	DoctorCmd, ExportCmd, Print                                        bool
	Url, Dir                                                           string
}

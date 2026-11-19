// Package doctor answers one question without changing anything: can this machine run this
// project? It looks at the tools each stack needs, Docker and the services the project
// uses, the ports its apps want, and free disk space. It runs only version commands of
// tools, never code from the project.
package doctor

package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/netip"
	"os"
	"strings"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
)

const label = "devdb"

func newClient() (*client.Client, error) {
	// if DOCKER_HOST is set, use it
	if os.Getenv("DOCKER_HOST") != "" {
		return client.New(client.FromEnv)
	}

	// try common socket paths in order
	sockets := []string{
		// Docker
		"/var/run/docker.sock",
		// Podman rootless
		fmt.Sprintf("/run/user/%d/podman/podman.sock", os.Getuid()),
		// Podman rootful
		"/run/podman/podman.sock",
	}

	for _, sock := range sockets {
		connnection, err := net.Dial("unix", sock)
		if err == nil {
			connnection.Close()
			return client.New(client.WithHost("unix://" + sock))
		}
	}

	return nil, fmt.Errorf("no container runtime found — is Docker or Podman running?")
}

func pullImage(ctx context.Context, cli *client.Client, ref string) error {
	fmt.Fprintf(os.Stderr, "pulling %s...\n", ref)
	out, err := cli.ImagePull(ctx, ref, client.ImagePullOptions{})
	if err != nil {
		return err
	}
	defer out.Close()
	io.Copy(io.Discard, out)
	return nil
}

func killExisting(
	ctx context.Context,
	cli *client.Client,
	name string,
	port int,
) error {
	f := client.Filters{}.Add("label", label)
	containers, err := cli.ContainerList(ctx, client.ContainerListOptions{
		All:     true,
		Filters: f,
	})
	if err != nil {
		return err
	}
	for _, c := range containers.Items {
		// always kill by name match
		if c.Labels[label] == name {
			kill(ctx, cli, c.ID)
			continue
		}
		// also kill anything on the same port
		for _, p := range c.Ports {
			if int(p.PublicPort) == port {
				kill(ctx, cli, c.ID)
				break
			}
		}
	}
	return nil
}

func kill(ctx context.Context, cli *client.Client, id string) {
	fmt.Fprintf(os.Stderr, "removing existing container %s...\n", id[:12])
	cli.ContainerStop(ctx, id, client.ContainerStopOptions{})
	cli.ContainerRemove(ctx, id, client.ContainerRemoveOptions{
		Force:         true,
		RemoveVolumes: true,
	})
}

func listContainers(ctx context.Context, cli *client.Client) error {
	f := client.Filters{}.Add("label", label)
	f = f.Add("label", label)
	containers, err := cli.ContainerList(ctx, client.ContainerListOptions{
		All:     true,
		Filters: f,
	})
	if err != nil {
		return err
	}
	for _, c := range containers.Items {
		fmt.Printf("%-20s %-15s %s\n", c.Labels[label], c.Image, c.State)
	}
	return nil
}

func runContainer(
	context context.Context,
	cli *client.Client,
	config RunConfig,
	profile DBProfile,
) error {
	name := generateName(config)

	if profile.defaults != nil {
		config = profile.defaults(config)
		name = config.name
	}

	if profile.validate != nil {
		if err := profile.validate(config); err != nil {
			return err
		}
	}

	if config.host == "" {
		config.host = "localhost"
	}
	if config.port == 0 {
		config.port = profile.port
	}
	if config.user == "" {
		config.user = name
	}
	if config.pass == "" {
		config.pass = name
	}

	version := config.version
	if version == "" {
		version = profile.latest
	}
	tag, ok := profile.versions[version]
	if !ok {
		return fmt.Errorf("unsupported version %q for %s", version, config.db)
	}
	ref := profile.image + ":" + tag

	if err := killExisting(context, cli, name, config.port); err != nil {
		return err
	}
	if err := pullImage(context, cli, ref); err != nil {
		return err
	}
	port_str := fmt.Sprintf("%d/tcp", profile.port)
	port, err := network.ParsePort(port_str)
	if err != nil {
		return fmt.Errorf("invalid port %s: %w", port_str, err)
	}
	result, err := cli.ContainerCreate(context, client.ContainerCreateOptions{
		Name: "devdb-" + name,
		Config: &container.Config{
			Image:    ref,
			Hostname: "devdb-" + name,
			Env:      toEnvSlice(profile.env(config)),
			Labels: map[string]string{
				label: name,
			},
		},

		HostConfig: &container.HostConfig{
			Privileged: profile.privileged,
			Binds:      makeVolumes(profile.volumes, "devdb-"+name),
			PortBindings: network.PortMap{
				network.Port(port): []network.PortBinding{{
					HostIP:   netip.MustParseAddr("127.0.0.1"),
					HostPort: fmt.Sprintf("%d", config.port)}},
			},
		},
	})
	if err != nil {
		return err
	}

	_, err = cli.ContainerStart(context, result.ID, client.ContainerStartOptions{})
	if err != nil {
		return err
	}

	if err := ready(context, cli, result.ID, config, profile); err != nil {
		return err
	}

	fmt.Println(renderDSN(profile.dsn, config))
	return nil
}

func toEnvSlice(env map[string]string) []string {
	out := make([]string, 0, len(env))
	for k, v := range env {
		out = append(out, k+"="+v)
	}
	return out
}

func makeVolumes(volumes []string, containerName string) []string {
	out := make([]string, 0, len(volumes))
	for _, v := range volumes {
		volName := containerName + strings.ReplaceAll(v, "/", "-")
		out = append(out, volName+":"+v)
	}
	return out
}

func ready(
	context context.Context,
	cli *client.Client,
	containerID string,
	config RunConfig,
	profile DBProfile,
) error {
	deadline := time.Now().Add(profile.readyin)
	address := fmt.Sprintf("%s:%d", config.host, config.port)

	// phase 1: wait for port
	fmt.Fprintf(os.Stderr, "waiting for %s...\n", address)
	for time.Now().Before(deadline) {
		remaining := time.Until(deadline).Round(time.Second)
		fmt.Fprintf(os.Stderr, "waiting for %s... (%s left)\n", address, remaining)
		conn, err := net.DialTimeout("tcp", address, 1*time.Second)
		if err == nil {
			conn.Close()
			break
		}
		time.Sleep(1 * time.Second)
	}
	if time.Now().After(deadline) {
		return fmt.Errorf("timed out waiting for %s", address)
	}

	// phase 2: readyCheck exec if defined
	if profile.readyon == nil {
		return nil
	}
	command := profile.readyon(config)
	fmt.Fprintf(os.Stderr, "waiting for database to be ready...\n")
	for time.Now().Before(deadline) {
		left := time.Until(deadline).Round(time.Second)
		fmt.Fprintf(os.Stderr, "waiting for database to be ready... (%s left)\n", left)
		exit, err := execInContainer(context, cli, containerID, command)
		if err == nil && exit == 0 {
			return nil
		}
		time.Sleep(5 * time.Second)
	}
	dumpLogs(context, cli, containerID)
	return fmt.Errorf("timed out waiting for database to be ready")
}

func dumpLogs(context context.Context, cli *client.Client, containerID string) {
	fmt.Fprintln(os.Stderr, "--- container logs ---")
	result, err := cli.ContainerLogs(context, containerID, client.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       "50",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not get logs: %v\n", err)
		return
	}
	defer result.Close()
	io.Copy(os.Stderr, result)
	fmt.Fprintln(os.Stderr, "--- end logs ---")
}

func execInContainer(
	ctx context.Context,
	cli *client.Client,
	containerID string,
	command []string,
) (int, error) {
	exec, err := cli.ExecCreate(ctx, containerID, client.ExecCreateOptions{
		Cmd:          command,
		AttachStdout: false,
		AttachStderr: false,
	})
	if err != nil {
		return -1, err
	}
	_, err = cli.ExecStart(ctx, exec.ID, client.ExecStartOptions{
		Detach: true,
	})
	if err != nil {
		return -1, err
	}
	// poll with a per-attempt timeout
	timeout := time.Now().Add(10 * time.Second)
	for time.Now().Before(timeout) {
		inspect, err := cli.ExecInspect(ctx, exec.ID, client.ExecInspectOptions{})
		if err != nil {
			return -1, err
		}
		if !inspect.Running {
			return inspect.ExitCode, nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return -1, fmt.Errorf("exec timed out")
}

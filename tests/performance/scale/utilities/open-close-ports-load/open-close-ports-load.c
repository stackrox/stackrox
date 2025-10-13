#include <netinet/in.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/socket.h>
#include <unistd.h>
#include <string.h>
#include <time.h>

int open_port(int port) {
	int server_fd;
	struct sockaddr_in address;
	int opt = 1;

	if ((server_fd = socket(AF_INET, SOCK_STREAM, 0)) < 0) {
		perror("socket failed");
		exit(EXIT_FAILURE);
	}

	if (setsockopt(server_fd, SOL_SOCKET,
				SO_REUSEADDR | SO_REUSEPORT, &opt,
				sizeof(opt))) {
		perror("setsockopt");
		exit(EXIT_FAILURE);
	}
	address.sin_family = AF_INET;
	address.sin_addr.s_addr = INADDR_ANY;
	address.sin_port = htons(port);

	if (bind(server_fd, (struct sockaddr*)&address,
			sizeof(address))
		< 0) {
		perror("bind failed");
		exit(EXIT_FAILURE);
	}
	if (listen(server_fd, 3) < 0) {
		perror("listen");
		exit(EXIT_FAILURE);
	}

	return server_fd;
}

void open_and_close_ports(int start_port, int end_port, float num_per_second) {
    int nports = end_port - start_port + 1;
    int *server_fds = NULL;
    float sleep_time = 1.0 / num_per_second;

    server_fds = malloc(nports * sizeof(int));

    printf("sleep_time = %f\n", sleep_time);
    printf("start_port = %i\n", start_port);
    printf("end_port = %i\n", end_port);

    struct timespec start_time, end_time;
    struct timespec func_start_time, func_end_time;
    double elapsed_time;

    clock_gettime(CLOCK_MONOTONIC, &func_start_time);

    for (int port = start_port; port <= end_port; port++) {
	clock_gettime(CLOCK_MONOTONIC, &start_time);

        server_fds[port - start_port] = open_port(port);

	clock_gettime(CLOCK_MONOTONIC, &end_time);
	elapsed_time = ((double)(end_time.tv_sec - start_time.tv_sec)) +
                       ((double)(end_time.tv_nsec - start_time.tv_nsec)) / 1.0e9;

        if (elapsed_time < sleep_time) {
            unsigned int curr_sleep_time = ((unsigned int)((sleep_time - elapsed_time) * 1000000.0));
            usleep(curr_sleep_time);
        }
    }
    clock_gettime(CLOCK_MONOTONIC, &func_end_time);
    elapsed_time = ((double)(func_end_time.tv_sec - func_start_time.tv_sec)) +
                   ((double)(func_end_time.tv_nsec - func_start_time.tv_nsec)) / 1.0e9;

    float real_num_per_second = (float)nports / elapsed_time;

    printf("nports = %i\n", nports);
    printf("elapsed_time = %f\n", elapsed_time);
    printf("real_num_per_second = %f\n", real_num_per_second);
    printf("Closing ports");

    for (int port = start_port; port <= end_port; port++) {
	clock_gettime(CLOCK_MONOTONIC, &start_time);

	if (server_fds[port - start_port] != -1) {
		close(server_fds[port - start_port]);
	}

	clock_gettime(CLOCK_MONOTONIC, &end_time);
	elapsed_time = ((double)(end_time.tv_sec - start_time.tv_sec)) +
                       ((double)(end_time.tv_nsec - start_time.tv_nsec)) / 1.0e9;

        if (elapsed_time < sleep_time) {
            unsigned int curr_sleep_time = ((unsigned int)((sleep_time - elapsed_time) * 1000000.0));
            usleep(curr_sleep_time);
        }
    }

}

int main(int argc, char *argv[]) {

    if (argc != 4) {
        fprintf(stderr, "Usage: %s <startPort> <endPort> <numPerSecond>\n", argv[0]);
        return 1;
    }

    int start_port = atoi(argv[1]);
    int end_port = atoi(argv[2]);
    float num_per_second = atof(argv[3]);

    open_and_close_ports(start_port, end_port, num_per_second);

    return 0;
}

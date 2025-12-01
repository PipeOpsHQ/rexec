FROM debian:bookworm

# Set environment variables
ENV DEBIAN_FRONTEND=noninteractive
ENV TERM=xterm-256color
ENV LANG=C.UTF-8
ENV LC_ALL=C.UTF-8

# Install essential packages including SSH server
RUN apt-get update && apt-get install -y \
    bash \
    curl \
    wget \
    git \
    vim \
    nano \
    htop \
    tree \
    sudo \
    locales \
    ca-certificates \
    gnupg \
    lsb-release \
    build-essential \
    python3 \
    python3-pip \
    nodejs \
    npm \
    openssh-server \
    openssh-client \
    net-tools \
    iputils-ping \
    dnsutils \
    procps \
    less \
    man-db \
    zip \
    unzip \
    jq \
    && rm -rf /var/lib/apt/lists/*

# Configure SSH server
RUN mkdir -p /var/run/sshd \
    && mkdir -p /etc/ssh/sshd_config.d \
    && echo "Port 22" > /etc/ssh/sshd_config.d/rexec.conf \
    && echo "PermitRootLogin no" >> /etc/ssh/sshd_config.d/rexec.conf \
    && echo "PasswordAuthentication no" >> /etc/ssh/sshd_config.d/rexec.conf \
    && echo "PubkeyAuthentication yes" >> /etc/ssh/sshd_config.d/rexec.conf \
    && echo "AuthorizedKeysFile .ssh/authorized_keys" >> /etc/ssh/sshd_config.d/rexec.conf \
    && echo "X11Forwarding no" >> /etc/ssh/sshd_config.d/rexec.conf \
    && echo "AllowAgentForwarding yes" >> /etc/ssh/sshd_config.d/rexec.conf \
    && echo "AllowTcpForwarding yes" >> /etc/ssh/sshd_config.d/rexec.conf \
    && echo "PrintMotd no" >> /etc/ssh/sshd_config.d/rexec.conf \
    && echo "AcceptEnv LANG LC_*" >> /etc/ssh/sshd_config.d/rexec.conf \
    && ssh-keygen -A

# Create a non-root user
RUN useradd -m -s /bin/bash -G sudo user \
    && echo "user ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers

# Set up SSH directory for user
RUN mkdir -p /home/user/.ssh \
    && chmod 700 /home/user/.ssh \
    && touch /home/user/.ssh/authorized_keys \
    && chmod 600 /home/user/.ssh/authorized_keys \
    && chown -R user:user /home/user/.ssh

# Create workspace directory
RUN mkdir -p /home/user/workspace && chown -R user:user /home/user

# Set working directory
WORKDIR /home/user

# Configure bash
RUN echo 'export PS1="\[\033[01;32m\]\u@rexec\[\033[00m\]:\[\033[01;34m\]\w\[\033[00m\]\$ "' >> /home/user/.bashrc \
    && echo 'alias ll="ls -la"' >> /home/user/.bashrc \
    && echo 'alias la="ls -A"' >> /home/user/.bashrc \
    && chown user:user /home/user/.bashrc

# Create entrypoint script that starts SSH and keeps container running
RUN echo '#!/bin/bash\n\
    # Start SSH server\n\
    sudo /usr/sbin/sshd\n\
    \n\
    # Execute passed command or start shell\n\
    if [ $# -eq 0 ]; then\n\
    exec /bin/bash\n\
    else\n\
    exec "$@"\n\
    fi' > /entrypoint.sh \
    && chmod +x /entrypoint.sh

# Expose SSH port
EXPOSE 22

# Switch to non-root user
USER user

# Default entrypoint
ENTRYPOINT ["/entrypoint.sh"]
CMD ["/bin/bash"]

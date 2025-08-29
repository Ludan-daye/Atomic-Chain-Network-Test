# NetCrate Homebrew Formula
# This formula installs NetCrate from GitHub releases

class Netcrate < Formula
  desc "Network security testing toolkit with compliance controls"
  homepage "https://github.com/netcrate/netcrate"
  version "1.0.0"
  
  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/netcrate/netcrate/releases/download/v#{version}/netcrate_Darwin_arm64.tar.gz"
      sha256 "" # This will be automatically updated by GoReleaser
    else
      url "https://github.com/netcrate/netcrate/releases/download/v#{version}/netcrate_Darwin_x86_64.tar.gz"
      sha256 "" # This will be automatically updated by GoReleaser
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/netcrate/netcrate/releases/download/v#{version}/netcrate_Linux_arm64.tar.gz"
      sha256 "" # This will be automatically updated by GoReleaser
    else
      url "https://github.com/netcrate/netcrate/releases/download/v#{version}/netcrate_Linux_x86_64.tar.gz"
      sha256 "" # This will be automatically updated by GoReleaser
    end
  end

  license "MIT"
  head "https://github.com/netcrate/netcrate.git", branch: "main"

  # Dependencies
  depends_on "nmap" => :optional

  def install
    bin.install "netcrate"
    
    # Install documentation
    doc.install "README.md" if File.exist?("README.md")
    doc.install "CHANGELOG.md" if File.exist?("CHANGELOG.md")
    
    # Install templates if they exist
    if Dir.exist?("templates")
      (share/"netcrate").install "templates"
    end
    
    # Install man pages if they exist
    if Dir.exist?("docs/man")
      man1.install Dir["docs/man/*.1"]
    end
    
    # Generate shell completions
    generate_completions_from_executable(bin/"netcrate", "completion")
  end

  def post_install
    # Create config directory
    (var/"netcrate").mkpath
    
    # Set appropriate permissions
    (var/"netcrate").chmod(0755)
    
    ohai "NetCrate installed successfully!"
    ohai "Config directory: #{var}/netcrate"
    ohai "To get started, run: netcrate --help"
  end

  test do
    # Test that the binary works and shows version
    system "#{bin}/netcrate", "--version"
    assert_match version.to_s, shell_output("#{bin}/netcrate --version")
    
    # Test basic help functionality
    system "#{bin}/netcrate", "--help"
    
    # Test config initialization
    system "#{bin}/netcrate", "config", "show"
  end

  def caveats
    <<~EOS
      NetCrate is a network security testing toolkit.
      
      IMPORTANT: Only use NetCrate on networks you own or have explicit 
      permission to test. Unauthorized network scanning may violate laws 
      and policies.
      
      For enhanced functionality, you may want to install optional dependencies:
        brew install nmap
      
      Some features require elevated privileges:
        sudo netcrate ops discover 192.168.1.0/24
      
      Configuration is stored in ~/.netcrate/
      
      Get started with:
        netcrate quick              # Auto-detect and scan local network  
        netcrate config rate list   # View scanning speed presets
        netcrate --help             # Show all available commands
        
      For more information, visit: https://github.com/netcrate/netcrate
    EOS
  end
end
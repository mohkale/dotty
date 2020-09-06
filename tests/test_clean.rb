require_relative './utils'

RSpec.describe :clean do
  it "can clean out dead links" do
    dotty = Dotty.new
    dest = Pathname.new(dotty.install_dir) / "foo"
    src = Pathname.new(dotty.config_dir) / "foo"
    dest.make_symlink(src)
    expect { src.lstat }.to raise_error(Errno::ENOENT)
    expect { dest.lstat }.to_not raise_error

    dotty_run_script '((:clean "~/"))' do |dotty|
      # dotty failed to remove broken link
      expect { dest.lstat }.to raise_error(Errno::ENOENT)
    end
  ensure
    dotty.cleanup
  end

  it "can clean out dead links recursively" do
    dotty = Dotty.new
    dest = Pathname.new(dotty.install_dir) / "foo" / "bar"
    src = Pathname.new(dotty.config_dir) / "bar"
    dest.parent.mkdir()
    dest.make_symlink(src)
    expect { src.lstat }.to raise_error(Errno::ENOENT)
    expect { dest.lstat }.to_not raise_error

    dotty_run_script '((:clean {:path "~/" :recursive true}))' do |dotty|
      expect { dest.lstat }.to raise_error(Errno::ENOENT)
      # parent of cleaned directories should still exist
      expect(dest.parent).to exist
    end
  ensure
    dotty.cleanup
  end

  it "doesn't clean valid links" do
    dotty = Dotty.new
    src = Pathname.new(dotty.config_dir) / "foo"
    dest = Pathname.new(dotty.install_dir) / "foo"
    src.open("w")
    dest.make_symlink(src)
    expect { src.lstat }.to_not raise_error
    expect { dest.lstat }.to_not raise_error

    dotty_run_script '((:clean "~/"))' do |dotty|
      expect { src.lstat }.to_not raise_error
      expect { dest.lstat }.to_not raise_error
    end
  ensure
    dotty.cleanup
  end

  it "only cleans links pointing to dotfiles" do
    dotty = Dotty.new
    # technically a path in my home directory is not in my
    # dotfiles directory.
    src = Pathname.new(dotty.install_dir) / "foo"
    dest = Pathname.new(dotty.install_dir) / "bar"
    dest.make_symlink(src)
    expect { src.lstat }.to raise_error(Errno::ENOENT)
    expect { dest.lstat }.to_not raise_error

    dotty_run_script '((:clean "~/"))' do |dotty|
      expect { src.lstat }.to raise_error(Errno::ENOENT)
      expect { dest.lstat }.to_not raise_error
    end
  ensure
    dotty.cleanup
  end

  it "cleans any broken links (when force is true)" do
    dotty = Dotty.new
    src = Pathname.new(dotty.install_dir) / "foo"
    dest = Pathname.new(dotty.install_dir) / "bar"
    dest.make_symlink(src)
    expect { src.lstat }.to raise_error(Errno::ENOENT)
    expect { dest.lstat }.to_not raise_error

    dotty_run_script '((:clean {:path "~/" :force true}))' do |dotty, _,_,err|
      expect { src.lstat }.to raise_error(Errno::ENOENT)
      expect { dest.lstat }.to raise_error(Errno::ENOENT)
    end
  ensure
    dotty.cleanup
  end
end

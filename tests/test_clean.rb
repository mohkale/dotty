# frozen_string_literal: true

require_relative './utils'

RSpec.describe :clean do
  dotty = Dotty.new

  it 'can clean out dead links' do
    dest = Pathname.new(dotty.install_dir) / 'foo'
    src = Pathname.new(dotty.config_dir) / 'foo'
    dest.make_symlink(src)
    expect { src.lstat }.to raise_error(Errno::ENOENT)
    expect { dest.lstat }.to_not raise_error

    dotty_run_script '((:clean "~/"))' do |_dotty|
      # dotty failed to remove broken link
      expect { dest.lstat }.to raise_error(Errno::ENOENT)
    end
  ensure
    dotty.cleanup
  end

  it 'can clean out dead links recursively' do
    dest = Pathname.new(dotty.install_dir) / 'foo' / 'bar'
    src = Pathname.new(dotty.config_dir) / 'bar'
    dest.parent.mkdir
    dest.make_symlink(src)
    expect { src.lstat }.to raise_error(Errno::ENOENT)
    expect { dest.lstat }.to_not raise_error

    dotty_run_script '((:clean {:path "~/" :recursive true}))' do |_dotty|
      expect { dest.lstat }.to raise_error(Errno::ENOENT)
      # parent of cleaned directories should still exist
      expect(dest.parent).to exist
    end
  ensure
    dotty.cleanup
  end

  it "doesn't clean valid links" do
    src = Pathname.new(dotty.config_dir) / 'foo'
    dest = Pathname.new(dotty.install_dir) / 'foo'
    src.open('w')
    dest.make_symlink(src)
    expect { src.lstat }.to_not raise_error
    expect { dest.lstat }.to_not raise_error

    dotty_run_script '((:clean "~/"))' do |_dotty|
      expect { src.lstat }.to_not raise_error
      expect { dest.lstat }.to_not raise_error
    end
  ensure
    dotty.cleanup
  end

  it 'only cleans links pointing to dotfiles' do
    # technically a path in my home directory is not in my
    # dotfiles directory.
    src = Pathname.new(dotty.install_dir) / 'foo'
    dest = Pathname.new(dotty.install_dir) / 'bar'
    dest.make_symlink(src)
    expect { src.lstat }.to raise_error(Errno::ENOENT)
    expect { dest.lstat }.to_not raise_error

    dotty_run_script '((:clean "~/"))' do |_dotty|
      expect { src.lstat }.to raise_error(Errno::ENOENT)
      expect { dest.lstat }.to_not raise_error
    end
  ensure
    dotty.cleanup
  end

  it 'cleans any broken links (when force is true)' do
    src = Pathname.new(dotty.install_dir) / 'foo'
    dest = Pathname.new(dotty.install_dir) / 'bar'
    dest.make_symlink(src)
    expect { src.lstat }.to raise_error(Errno::ENOENT)
    expect { dest.lstat }.to_not raise_error

    dotty_run_script '((:clean {:path "~/" :force true}))' do |_dotty, _, _, _err|
      expect { src.lstat }.to raise_error(Errno::ENOENT)
      expect { dest.lstat }.to raise_error(Errno::ENOENT)
    end
  ensure
    dotty.cleanup
  end
end

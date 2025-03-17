#compdef elles

# Completion for "elles"; https://github.com/arp242/elles
#
# Save as "_elles" in any directory in $fpath; see the current list with:
#
#    print -l $fpath
#
# To add your own directory (before compinit):
#
#   fpath=(~/.zsh/funcs $fpath)

local arguments

arguments=(
	'(-a --all)'{-a,--all}'[list entries starting with .]'
	'(-d --directory)'{-d,--directory}'[list directories themselves, instead of contents]'
	'(-H)'-H'[follow symlink on the command line]'
	'(-R --recursive)'{-R,-recursive}'[list subdirectories recursively]'
	'(-i --inore)'{-i,--inode}'[print inode numbers]'
	'(-g -groupname)'{-g,--groupname}'[always print group name]'

	'(-j --json)'{-j,--json}'[print as JSON]'
	'(-1 -C)'-l'[long listing]'
	'(-1 -C)'-ll'[longer listing]'
	'(-l -C -ll)'-1'[single column output]'
	'(-1 -l -ll)'-C'[columnar output]'
	'(--group-dirs)'--group-dirs'[group drectories first]'
	'(-n)'-n'[numeric uid and gid]'
	'(-L)'-L"[don't show symlink targets in -l]"
	'(-w --width)'{-w,--width}'[maximum column width]'
	'(--trim --no-trim)'--trim"[trim pathnames if they're too long to fit on the screen]"
	'(--no-trim --trim)'--no-trim'[disable --trim]'
	'(-o --octal)'{-o,--octal}'[file permissions as octal]'

	'--color=-[control use of color]:color:(never always auto)'
	'--hyperlink=[output terminal codes to link files using file::// URI]::when:(none auto always)'
	'(-p -F)'-p'[append / to directories]'
	'(-F -p)'-F'[append file type indicators]'
	'(-,)'-,'[print file sizes with thousands separators]'
	'--blocks=-[format for file sizes]:block:(1 s S K M G)'
	'(-c -u)'-c'[use creation (btime) in -l and -t sorting]'
	'(-c -u)'-u'[use access in -l and -t sorting]'
	'(-T)'-T'[display full time info]'
	'(-TT)'-TT'[display full time info with nanoseconds and TZ]'
	'(-Q)'-Q'[quote paths with special shell characters or spaces]'
	'(-QQ)'-QQ'[quote all paths]'

	'(-r --reverse)'{-r,--reverse}'[reverse sort order]'
	'(--sort -t -U -v -X -W)-S[sort by size]'
	'(--sort -S -t -U -v -W)-X[sort by extension]'
	'(--sort -S -t -U -X -W)-v[sort by version (filename treated numerically)]'
	'(--sort -S -U -v -X -W)-t[sort by time]'
	'(--sort -S -U -v -X -t)-W[sort by width]'
	'(-S -t -U -v -X -W)--sort=[specify sort key]:sort key:(size time none version extension width)'

	'(- :)--help[display help information]'
	'(- :)--version[display version information]'

	'*:file:_files'
)

_arguments -s -S : $arguments

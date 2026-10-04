import 'package:flutter/material.dart';

import '../../theme/app_theme.dart';

/// What the user chose in the remove-container confirmation.
class RemoveContainerChoice {
  /// Also delete the containers' anonymous volumes (`docker rm -v`). Named
  /// volumes are always kept.
  final bool removeVolumes;

  const RemoveContainerChoice({required this.removeVolumes});
}

/// Confirms removing [count] container(s). Pops with the user's choice, or
/// null if cancelled. [stopsRunning] adds the note that running containers
/// are stopped first (every caller removes with force today).
Future<RemoveContainerChoice?> showRemoveContainerDialog(
  BuildContext context, {
  int count = 1,
  bool stopsRunning = true,
}) {
  return showDialog<RemoveContainerChoice>(
    context: context,
    builder: (_) =>
        _RemoveContainerDialog(count: count, stopsRunning: stopsRunning),
  );
}

class _RemoveContainerDialog extends StatefulWidget {
  final int count;
  final bool stopsRunning;

  const _RemoveContainerDialog({
    required this.count,
    required this.stopsRunning,
  });

  @override
  State<_RemoveContainerDialog> createState() => _RemoveContainerDialogState();
}

class _RemoveContainerDialogState extends State<_RemoveContainerDialog> {
  // On by default: once its container is gone, an anonymous volume can't be
  // reattached by name and only takes up disk.
  bool _removeVolumes = true;

  @override
  Widget build(BuildContext context) {
    final plural = widget.count != 1;
    return AlertDialog(
      title: Text(
        plural ? 'Remove ${widget.count} containers?' : 'Remove container?',
      ),
      content: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            'This permanently deletes ${plural ? 'them' : 'the container'}'
            '${widget.stopsRunning ? ', stopping ${plural ? 'any that are' : 'it first if'} running' : ''}. '
            'Any data outside a named volume is lost.',
          ),
          const SizedBox(height: 8),
          CheckboxListTile(
            contentPadding: EdgeInsets.zero,
            controlAffinity: ListTileControlAffinity.leading,
            value: _removeVolumes,
            onChanged: (v) => setState(() => _removeVolumes = v ?? false),
            title: const Text('Also remove anonymous volumes'),
            subtitle: const Text(
              'Frees the disk space they use. Named volumes are kept.',
            ),
          ),
        ],
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.of(context).pop(),
          child: const Text('Cancel'),
        ),
        FilledButton(
          style: FilledButton.styleFrom(backgroundColor: AppColors.failed),
          onPressed: () => Navigator.of(
            context,
          ).pop(RemoveContainerChoice(removeVolumes: _removeVolumes)),
          child: const Text('Remove'),
        ),
      ],
    );
  }
}
